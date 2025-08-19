// Command zoomwebhookd runs the Zoom webhook ingestion service.
//
// It exposes a fast HTTP(S) endpoint that accepts Zoom webhooks, verifies
// signatures off the hot path, batches valid events, and writes them to
// pluggable outputs (file w/ rotation and/or HTTP sinks like Splunk HEC).
//
// Configuration:
//   - YAML: see config.yaml (paths, shards, queues, TLS, outputs, metrics, tracing)
//   - Secrets: never in YAML. Zoom's secret and HEC tokens are read from env.
//
// Observability:
//   - Logging: Logrus (JSON), suitable for Loki
//   - Metrics: Prometheus /metrics endpoint
//   - Tracing: OpenTelemetry → OTLP (Tempo)
package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"

	"github.com/mejooo/webhook_engine/internal/fastpath"
	"github.com/mejooo/webhook_engine/internal/server"
	"github.com/mejooo/webhook_engine/internal/zoomapp"
	"github.com/mejooo/webhook_engine/pkg/outputs"
	"github.com/mejooo/webhook_engine/pkg/tracing"
)

func main() {
	// ---- Flags -----------------------------------------------------------------
	var cfgPath string
	flag.StringVar(&cfgPath, "config", "config.yaml", "Path to YAML configuration")
	flag.Parse()

	// ---- Load config -----------------------------------------------------------
	cfg, err := server.LoadConfig(cfgPath)
	if err != nil {
		// Use a temporary console logger if config couldn't be loaded.
		logrus.Fatalf("config: %v", err)
	}

	// ---- Logger (Logrus) -------------------------------------------------------
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	if lvl, err := logrus.ParseLevel(cfg.Logging.Level); err == nil {
		log.SetLevel(lvl)
	}
	if cfg.Logging.File != "" {
		f, ferr := os.OpenFile(cfg.Logging.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if ferr == nil {
			log.SetOutput(f)
		} else {
			log.WithError(ferr).Warn("failed to open log file, falling back to stdout")
		}
	}
	log.WithField("config", cfgPath).Info("configuration loaded")

	// ---- Tracing (Tempo via OTLP) ---------------------------------------------
	shutdownTracer, terr := tracing.Init("zoom-webhook", cfg.Tracing.OTLPEndpoint, cfg.Tracing.SampleRatio)
	if terr != nil {
		log.WithError(terr).Warn("tracing init failed (continuing without exporter)")
	}
	defer func() {
		_ = shutdownTracer(context.Background())
	}()

	// ---- Metrics server (Prometheus) ------------------------------------------
	if cfg.Metrics.PrometheusListen != "" {
		go func() {
			mux := http.NewServeMux()
			mux.Handle("/metrics", promhttp.Handler())
			srv := &http.Server{
				Addr:              cfg.Metrics.PrometheusListen,
				Handler:           mux,
				ReadHeaderTimeout: 5 * time.Second,
			}
			log.WithField("listen", cfg.Metrics.PrometheusListen).Info("metrics server started")
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.WithError(err).Error("metrics server exited")
			}
		}()
	}

	// ---- Secrets (from env only) ----------------------------------------------
	secret := os.Getenv(cfg.Zoom.SecretEnv)
	if secret == "" {
		log.Fatalf("missing env %s (Zoom webhook secret)", cfg.Zoom.SecretEnv)
	}
	token := []byte(secret)

	// ---- Outputs from config ---------------------------------------------------
	var drivers []outputs.Driver
	for _, oc := range cfg.Outputs {
		switch oc.Type {
		case "file":
			drivers = append(drivers, outputs.NewFile(
				log,
				oc.Path,
				oc.Rotation.MaxSizeMB,
				oc.Rotation.MaxBackups,
				oc.Rotation.MaxAgeDays,
				oc.Rotation.Compress,
			))
		case "http":
			drivers = append(drivers, outputs.NewHTTP(
				log,
				oc.URL,
				oc.TimeoutMS,
				oc.Parallel,
				oc.HECTokenEnv, // read at runtime via env if set
			))
		default:
			log.Fatalf("unknown output type: %q", oc.Type)
		}
	}
	outMgr := outputs.NewManager(log, drivers...)
	if err := outMgr.Start(context.Background()); err != nil {
		log.Fatalf("outputs start: %v", err)
	}
	defer outMgr.Stop(context.Background())

	// ---- Build shards / pipeline ----------------------------------------------
	shards := make([]*fastpath.Shard, cfg.Server.Shards)
	for i := 0; i < cfg.Server.Shards; i++ {
		r := fastpath.NewRing(cfg.Server.QueueSize)
		valOut := make(chan fastpath.Validated, cfg.Server.QueueSize)

		// Signature validators (Zoom v0 HMAC) per shard
		for v := 0; v < cfg.Server.ValidatorsPerShard; v++ {
			go (&fastpath.Validator{Token: token, In: r.C(), Out: valOut, Log: log}).Run()
		}

		// Batch writer → outputs (fan-out)
		bw := &fastpath.BatchWriter{
			In:        valOut,
			Sink:      outMgr,
			MaxN:      cfg.Server.Batch.Size,
			Linger:    time.Duration(cfg.Server.Batch.LingerMS) * time.Millisecond,
			TimeoutMS: 2000,
			Log:       log,
		}
		go bw.Run(context.Background())

		shards[i] = &fastpath.Shard{ID: i, Ring: r, ValOut: valOut}
	}
	log.WithFields(logrus.Fields{
		"shards":     cfg.Server.Shards,
		"queue_size": cfg.Server.QueueSize,
		"validators": cfg.Server.ValidatorsPerShard,
	}).Info("pipeline initialized")

	// ---- App (HTTP routing + fast path) ---------------------------------------
	app := server.NewApp(cfg, log)
	app.AttachFastRings(shards)

	// Zoom CRC pre-handler config (keep as you define it)
	zcfg := zoomapp.Config{}
	handler := app.FastHandler(zcfg)

	srv := &fasthttp.Server{
		Handler:            handler,
		ReadTimeout:        5 * time.Second,
		WriteTimeout:       5 * time.Second,
		MaxRequestBodySize: 8 << 20, // 8 MiB; tune as needed
	}

	// ---- Serve HTTP or HTTPS based on config ----------------------------------
	go func() {
		if cfg.Server.TLS.Enabled {
			log.WithFields(logrus.Fields{
				"listen": cfg.Server.Listen,
				"cert":   cfg.Server.TLS.CertFile,
				"key":    cfg.Server.TLS.KeyFile,
			}).Info("serving HTTPS")
			if err := srv.ListenAndServeTLS(cfg.Server.Listen, cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil {
				log.WithError(err).Fatal("server exited")
			}
		} else {
			log.WithField("listen", cfg.Server.Listen).Info("serving HTTP")
			if err := srv.ListenAndServe(cfg.Server.Listen); err != nil {
				log.WithError(err).Fatal("server exited")
			}
		}
	}()

	// ---- Graceful shutdown on signals -----------------------------------------
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	s := <-sigc
	log.WithField("signal", s.String()).Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(); err != nil {
		log.WithError(err).Warn("server shutdown error")
	}

	// Give background goroutines a short moment to flush logs/batches.
	<-shutdownCtx.Done()
	log.Info("bye")
}
