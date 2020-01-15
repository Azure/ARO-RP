package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

const (
	defaultMaxWorkers      = 20
	defaultMaxDequeueCount = 5
)

type backend struct {
	baseLog *logrus.Entry
	env     env.Interface
	db      *database.Database
	m       metrics.Interface

	mu         sync.Mutex
	cond       *sync.Cond
	workers    int32
	maxWorkers *int32
	stopping   atomic.Value

	ocb *openShiftClusterBackend
	sb  *subscriptionBackend

	maxDequeueCount *int
}

// Runnable represents a runnable object
type Runnable interface {
	Run(context.Context, <-chan struct{})
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

	if len(os.Getenv("MAX_WORKERS")) > 0 {
		i, err := strconv.Atoi(os.Getenv("MAX_WORKERS"))
		if err != nil {
			return nil, err
		}
		b.maxWorkers = to.Int32Ptr(int32(i))
	}
	if len(os.Getenv("MAX_DEQUEUE")) > 0 {
		i, err := strconv.Atoi(os.Getenv("MAX_DEQUEUE"))
		if err != nil {
			return nil, err
		}
		b.maxDequeueCount = to.IntPtr(i)
	}

	return b, nil
}

func (b *backend) Run(ctx context.Context, stop <-chan struct{}) {
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
	if b.maxWorkers == nil {
		b.maxWorkers = to.Int32Ptr(defaultMaxWorkers)
	}
	if b.maxDequeueCount == nil {
		b.maxDequeueCount = to.IntPtr(defaultMaxDequeueCount)
	}

	for {
		b.mu.Lock()
		for atomic.LoadInt32(&b.workers) >= *b.maxWorkers && !b.stopping.Load().(bool) {
			b.cond.Wait()
		}
		b.mu.Unlock()

		if b.stopping.Load().(bool) {
			break
		}

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
