package outputs

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type Record struct{ Body []byte }

type Driver interface {
	Start(context.Context) error
	Stop(context.Context) error
	Write(context.Context, []Record) error
	Name() string
}

type Manager struct {
	drivers []Driver
	log     *logrus.Logger
}

func NewManager(log *logrus.Logger, ds ...Driver) *Manager {
	return &Manager{drivers: ds, log: log}
}

func (m *Manager) Start(ctx context.Context) error {
	for _, d := range m.drivers {
		if err := d.Start(ctx); err != nil {
			return fmt.Errorf("start %s: %w", d.Name(), err)
		}
		m.log.WithField("driver", d.Name()).Info("output started")
	}
	return nil
}
func (m *Manager) Stop(ctx context.Context) {
	for i := len(m.drivers) - 1; i >= 0; i-- {
		_ = m.drivers[i].Stop(ctx)
	}
}
func (m *Manager) FanOut(ctx context.Context, batch []Record) error {
	var wg sync.WaitGroup
	errc := make(chan error, len(m.drivers))
	for _, d := range m.drivers {
		wg.Add(1)
		go func(d Driver) {
			defer wg.Done()
			if err := d.Write(ctx, batch); err != nil {
				m.log.WithError(err).WithField("driver", d.Name()).Error("output write failed")
				errc <- err
			}
		}(d)
	}
	wg.Wait()
	close(errc)
	var first error
	for e := range errc {
		if first == nil {
			first = e
		}
	}
	return first
}
