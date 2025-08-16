package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	ReceivedTotal   = prometheus.NewCounter(prometheus.CounterOpts{Name: "webhook_received_total", Help: "events received"})
	ValidatedTotal  = prometheus.NewCounter(prometheus.CounterOpts{Name: "webhook_validated_total", Help: "events validated"})
	InvalidTotal    = prometheus.NewCounter(prometheus.CounterOpts{Name: "webhook_invalid_total", Help: "invalid events"})
	Dropped429      = prometheus.NewCounter(prometheus.CounterOpts{Name: "webhook_dropped_429_total", Help: "429 drops"})
	FastShardQueued = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "fast_shard_queued", Help: "queued per shard"}, []string{"shard"})
)

func RegisterAll() {
	prometheus.MustRegister(ReceivedTotal, ValidatedTotal, InvalidTotal, Dropped429, FastShardQueued)
}
