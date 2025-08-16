package fastpath

import (
	"encoding/binary"
	"math/rand"
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

type BatchWriter struct {
	DB     *badger.DB
	In     <-chan []byte
	MaxN   int
	Linger time.Duration
}

func (w *BatchWriter) Run() {
	wb := w.DB.NewWriteBatch(); defer wb.Cancel()
	timer := time.NewTimer(w.Linger); defer timer.Stop()

	n := 0
	for {
		select {
		case b, ok := <-w.In:
			if !ok {
				if n>0 { _ = wb.Flush() }
				return
			}
			k := make([]byte, 8); binary.LittleEndian.PutUint64(k, rand.Uint64())
			_ = wb.SetEntry(badger.NewEntry(k, b))
			n++
			if n >= w.MaxN {
				_ = wb.Flush(); n = 0
				if !timer.Stop() { select { case <-timer.C: default: } }
				timer.Reset(w.Linger)
			}
		case <-timer.C:
			if n>0 { _ = wb.Flush(); n = 0 }
			timer.Reset(w.Linger)
		}
	}
}
