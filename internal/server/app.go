package server

import (
	"bytes"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"github.com/mejooo/webhook_engine/internal/zoomapp"
	"github.com/mejooo/webhook_engine/internal/fastpath"
	"github.com/mejooo/webhook_engine/pkg/fastqueue"
	"github.com/mejooo/webhook_engine/pkg/metrics"
)

type App struct {
	Cfg  RootConfig
	Log  *logrus.Logger
	Fast struct{ Rings []*fastqueue.Ring }
}

func NewApp(cfg RootConfig, log *logrus.Logger) *App {
	metrics.RegisterAll()
	return &App{Cfg: cfg, Log: log}
}

// AttachFastRings wires the server to the actual shard rings.
// Accepts real fastpath shards, extract their Ring pointers.
func (a *App) AttachFastRings(shards []*fastpath.Shard) {
	a.Fast.Rings = make([]*fastqueue.Ring, len(shards))
	for i, s := range shards {
		a.Fast.Rings[i] = s.Ring
	}
}


// Handler is the regular non-fast handler (minimal in this sample).
func (a *App) Handler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		if bytes.Equal(ctx.Path(), []byte("/health")) {
			ctx.SetStatusCode(200); ctx.SetBodyString("ok"); return
		}
		ctx.SetStatusCode(404)
	}
}

// FastHandler wraps CRC pre-handler + fast-path for /webhook/zoom.
func (a *App) FastHandler(zcfg zoomapp.Config) fasthttp.RequestHandler {
	base := a.Handler()
	crc := zoomapp.ZoomPreHandler(a, zcfg)
	fast := func(ctx *fasthttp.RequestCtx) {
		// only for zoom path; otherwise fallback
		if !bytes.Equal(ctx.Path(), []byte("/webhook/zoom")) || !ctx.IsPost() {
			base(ctx); return
		}
		sig := append([]byte(nil), ctx.Request.Header.Peek("x-zm-signature")...)
		ts  := append([]byte(nil), ctx.Request.Header.Peek("x-zm-request-timestamp")...)
		if len(sig)==0 || len(ts)==0 { ctx.SetStatusCode(400); return }
		body := append([]byte(nil), ctx.PostBody()...)

		if len(a.Fast.Rings)==0 {
			ctx.SetStatusCode(503); return
		}
		shard := fastqueue.ShardFor(body, len(a.Fast.Rings))
		ok := a.Fast.Rings[shard].TryPush(fastqueue.Event{Body: body, Sig: sig, TS: ts})
		if !ok {
			metrics.Dropped429.Inc()
			ctx.Response.Header.Set("Retry-After", "1")
			ctx.SetStatusCode(429)
			return
		}
		metrics.ReceivedTotal.Inc()
		ctx.SetStatusCode(202)
	}
	return crc.Wrap(fast)
}

// JSON helper (not heavily used here)
func writeJSON(ctx *fasthttp.RequestCtx, v any, code int) {
	b, _ := json.Marshal(v)
	ctx.Response.Header.SetContentType("application/json")
	ctx.SetStatusCode(code)
	ctx.SetBody(b)
}
