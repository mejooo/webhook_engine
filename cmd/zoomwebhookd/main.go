package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
	"log"
	"strconv"

	"github.com/spf13/pflag"
	"github.com/valyala/fasthttp"
	"github.com/sirupsen/logrus"

	"webhook-engine/internal/server"
	"webhook-engine/internal/zoomapp"
	"webhook-engine/internal/fastpath"
	"webhook-engine/pkg/tracing"
	"webhook-engine/pkg/validators/zoom"
)

func die(err error) { if err != nil { log.Fatal(err) } }

func main() {
	var cfgPath string
	var fast bool
	var reuseport int
	var metricsTick int

	pflag.StringVar(&cfgPath, "config", "docker/configs/zoomapp.yaml", "config path")
	pflag.BoolVar(&fast, "fastpath", true, "enable fastpath")
	pflag.IntVar(&reuseport, "reuseport-listeners", 0, "SO_REUSEPORT listeners (linux)")
	pflag.IntVar(&metricsTick, "fast-metrics-ms", 500, "fastpath metrics tick ms")
	pflag.Parse()

	logr := logrus.New()
	logr.SetFormatter(&logrus.JSONFormatter{})
	logr.SetLevel(logrus.WarnLevel)

	rootCfg, err := server.LoadRootConfig(cfgPath)
	die(err)
	zcfg, err := zoomapp.Load(cfgPath)
	die(err)

	shutdownTracing, _ := tracing.Setup(tracing.Options{
		ServiceName:  "zoom-webhook",
		SampleRatio:  rootCfg.Tracing.SampleRatio,
		OTLPEndpoint: rootCfg.Tracing.OTLPEndpoint,
	})
	defer shutdownTracing(context.Background())

	app := server.NewApp(rootCfg, logr)

	// fastpath build
	var stopFast func() error
	if fast || zcfg.Fastpath.Enabled {
		secret, err := zoom.LoadSecretForCRC(rootCfg.Validators.Zoom.Secret)
		die(err)
		token := []byte(secret)
		shards, stop, err := fastpath.BuildShards(
			zcfg.Fastpath.Shards,
			zcfg.Fastpath.BaseDir,
			zcfg.Fastpath.RingSize,
			zcfg.Fastpath.ValidatorsPer,
			zcfg.Fastpath.BatchSize,
			time.Duration(zcfg.Fastpath.BatchLingerMS)*time.Millisecond,
			token,
		)
		die(err)
		stopFast = stop
		app.AttachFastRings(shards)

		// metrics ticker
		go func() {
			t := time.NewTicker(time.Duration(metricsTick)*time.Millisecond)
			defer t.Stop()
			for range t.C {
				fastpath.ReportShardMetrics(shards)
			}
		}()
	}

	handler := app.FastHandler(zcfg) // includes CRC pre-handler

	// graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigs; cancel(); if stopFast!=nil { _=stopFast() } }()

	// Serve: in Docker we use PORT=8080 (HTTP). TLS for bare metal not included in this sample.
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	srv := &fasthttp.Server{
		Handler: handler,
		ReadTimeout:  5*time.Second,
		WriteTimeout: 5*time.Second,
		Name: "zoomwebhookd",
	}
	logr.WithField("port", port).Warn("serving HTTP")
	if err := srv.ListenAndServe(":" + port); err != nil {
		logr.WithError(err).Error("server stopped")
	}
	_ = strconv.ErrRange // silence unused import on some toolchains
}
