package fastpath

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	"github.com/mejooo/webhook_engine/pkg/outputs"
	"github.com/mejooo/webhook_engine/pkg/metrics"
)

type BatchWriter struct {
	In        <-chan Validated
	Sink      *outputs.Manager
	MaxN      int
	Linger    time.Duration
	TimeoutMS int
	Log       *logrus.Logger
}

func (w *BatchWriter) Run(ctx context.Context) {
	tr := otel.Tracer("batch")
	buf := make([]outputs.Record, 0, w.MaxN)

	flush := func() {
		if len(buf) == 0 { return }
		_, span := tr.Start(ctx, "outputs.fanout")
		defer span.End()

		c, cancel := outputs.DeadlineCtx(ctx, w.TimeoutMS)
		if err := w.Sink.FanOut(c, buf); err != nil {
			metrics.OutputErrors.WithLabelValues("fanout").Inc()
			w.Log.WithError(err).Warn("fanout error")
		} else {
			metrics.BatchFlushTotal.Inc()
			metrics.BatchItems.Observe(float64(len(buf)))
			w.Log.WithField("items", len(buf)).Debug("flushed batch")
		}
		cancel()
		buf = buf[:0]
	}

	t := time.NewTicker(w.Linger)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			flush(); return
		case v, ok := <-w.In:
			if !ok { flush(); return }
			buf = append(buf, outputs.Record{Body: v.Body})
			if len(buf) >= w.MaxN { flush() }
		case <-t.C:
			flush()
		}
	}
}
