package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

const (
	maxWorkers      = 100
	maxDequeueCount = 5
)

type backend struct {
	baseLog *logrus.Entry
	env     env.Interface
	db      *database.Database
	m       metrics.Interface

	mu       sync.Mutex
	cond     *sync.Cond
	workers  int32
	stopping atomic.Value

	ocb *openShiftClusterBackend
	sb  *subscriptionBackend
}

// Runnable represents a runnable object
type Runnable interface {
	Run(context.Context, <-chan struct{}, chan<- struct{})
}

// NewBackend returns a new runnable backend
func NewBackend(ctx context.Context, log *logrus.Entry, env env.Interface, db *database.Database, m metrics.Interface) (Runnable, error) {
	b := &backend{
		baseLog: log,
		env:     env,
		db:      db,
		m:       m,
	}

	b.cond = sync.NewCond(&b.mu)
	b.stopping.Store(false)

	b.ocb = &openShiftClusterBackend{backend: b}
	b.sb = &subscriptionBackend{backend: b}

	return b, nil
}

func (b *backend) Run(ctx context.Context, stop <-chan struct{}, done chan<- struct{}) {
	defer recover.Panic(b.baseLog)

	t := time.NewTicker(time.Second)
	defer t.Stop()

	if stop != nil {
		go func() {
			defer recover.Panic(b.baseLog)

			<-stop
			b.baseLog.Print("stopping")
			b.stopping.Store(true)
			b.cond.Signal()
		}()
	}

	for {
		b.mu.Lock()
		for atomic.LoadInt32(&b.workers) >= maxWorkers && !b.stopping.Load().(bool) {
			b.cond.Wait()
		}
		b.mu.Unlock()

		if b.stopping.Load().(bool) {
			break
		} else {
			ocbDidWork, err := b.ocb.try(ctx)
			if err != nil {
				b.baseLog.Error(err)
			}

			sbDidWork, err := b.sb.try(ctx)
			if err != nil {
				b.baseLog.Error(err)
			}

			if !(ocbDidWork || sbDidWork) {
				<-t.C
			}
		}
	}

	for {
		b.mu.Lock()
		for atomic.LoadInt32(&b.workers) > 0 {
			b.cond.Wait()
		}
		b.mu.Unlock()
		break
	}
	close(done)
}
