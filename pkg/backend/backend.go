package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/env"
	"github.com/jim-minter/rp/pkg/util/recover"
)

const (
	maxWorkers      = 100
	maxDequeueCount = 5
)

type backend struct {
	baseLog *logrus.Entry
	env     env.Interface
	db      *database.Database

	mu       sync.Mutex
	cond     *sync.Cond
	workers  int32
	stopping atomic.Value

	ocb *openShiftClusterBackend
	sb  *subscriptionBackend
}

// Runnable represents a runnable object
type Runnable interface {
	Run(stop <-chan struct{})
}

// NewBackend returns a new runnable backend
func NewBackend(ctx context.Context, log *logrus.Entry, env env.Interface, db *database.Database) (Runnable, error) {
	b := &backend{
		baseLog: log,
		env:     env,
		db:      db,
	}

	b.cond = sync.NewCond(&b.mu)
	b.stopping.Store(false)

	b.ocb = &openShiftClusterBackend{backend: b}
	b.sb = &subscriptionBackend{backend: b}

	return b, nil
}

func (b *backend) Run(stop <-chan struct{}) {
	defer recover.Panic(b.baseLog)

	t := time.NewTicker(time.Second)
	defer t.Stop()

	go func() {
		defer recover.Panic(b.baseLog)

		<-stop
		b.baseLog.Print("stopping")
		b.stopping.Store(true)
		b.cond.Signal()
	}()

	for {
		b.mu.Lock()
		for atomic.LoadInt32(&b.workers) >= maxWorkers && !b.stopping.Load().(bool) {
			b.cond.Wait()
		}
		b.mu.Unlock()

		if b.stopping.Load().(bool) {
			break
		}

		ocbDidWork, err := b.ocb.try()
		if err != nil {
			b.baseLog.Error(err)
		}

		sbDidWork, err := b.sb.try()
		if err != nil {
			b.baseLog.Error(err)
		}

		if !(ocbDidWork || sbDidWork) {
			<-t.C
		}
	}
}
