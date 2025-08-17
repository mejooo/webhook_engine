package fastpath

import (
	"fmt"
	"path/filepath"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"

	"github.com/mejooo/webhook_engine/pkg/fastqueue"
	"github.com/mejooo/webhook_engine/pkg/metrics"
)

type Shard struct {
	ID		int
	Ring   *fastqueue.Ring
	ValOut chan []byte
	DB     *badger.DB
	done   chan struct{}
}

func OpenBadger(dir string) (*badger.DB, error) {
	opts := badger.DefaultOptions(dir).
		WithSyncWrites(true).
		WithCompression(options.None).
		WithValueLogFileSize(64 << 20).
		WithBlockCacheSize(16 << 20).
		WithIndexCacheSize(16 << 20)
	return badger.Open(opts)
}

func BuildShards(n int, baseDir string, ringSize, validatorsPer, batchSize int, linger time.Duration, token []byte) ([]*Shard, func() error, error) {
	if n <= 0 {
		n = 1
	}
	out := make([]*Shard, n)
	for i := 0; i < n; i++ {
		db, err := OpenBadger(filepath.Join(baseDir, fmt.Sprintf("shard-%02d", i)))
		if err != nil {
			return nil, nil, err
		}
		r := fastqueue.NewRing(ringSize)
		valOut := make(chan []byte, ringSize)

		for v := 0; v < validatorsPer; v++ {
			go (&Validator{Token: token, In: r.C(), Out: valOut}).Run()
		}

		bw := &BatchWriter{DB: db, In: valOut, MaxN: batchSize, Linger: linger}
		done := make(chan struct{})
		go func() { bw.Run(); close(done) }()

		out[i] = &Shard{Ring: r, ValOut: valOut, DB: db, done: done}
	}
	stop := func() error {
		var first error
		for _, s := range out {
			close(s.ValOut)
			<-s.done
			if err := s.DB.Close(); err != nil && first == nil {
				first = err
			}
		}
		return first
	}
	return out, stop, nil
}

func ReportShardMetrics(shards []*Shard) {
	total := 0
	for i, s := range shards {
		q := s.Ring.Len()
		metrics.FastShardQueued.WithLabelValues(fmt.Sprintf("%d", i)).Set(float64(q))
		total += q
	}
	_ = total
}
