package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/backend/openshiftcluster"
	"github.com/Azure/ARO-RP/pkg/backend/subscriptions"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

const (
	maxWorkers      = 100
	maxDequeueCount = 5
)

type backendManager struct {
	baseLog *logrus.Entry
	env     env.Interface
	db      *database.Database
	m       metrics.Interface

	mu       sync.Mutex
	cond     *sync.Cond
	stopping atomic.Value

	// registry contains all backends, registered to the manager
	registry []instance
}

type instance struct {
	name    string
	backend Backend
	workers int32
}

// Backend represents minimal interface for individual backends
type Backend interface {
	Handle(context.Context, *logrus.Entry, interface{}) error
	Dequeue() func(context.Context) (interface{}, error)
}

// Runnable represents a runnable object for all backends
type Runnable interface {
	Run(context.Context, <-chan struct{})
}

// NewBackendManager returns a new runnable backend manager object
func NewBackendManager(ctx context.Context, log *logrus.Entry, env env.Interface, db *database.Database, m metrics.Interface) (Runnable, error) {
	b := &backendManager{
		baseLog: log,
		env:     env,
		db:      db,
		m:       m,
	}

	b.cond = sync.NewCond(&b.mu)
	b.stopping.Store(false)

	// register backends to the manager for management
	b.registry = make([]instance, 2)
	b.registry = []instance{
		{
			name:    "openshiftcluster",
			backend: openshiftcluster.New(env, db, m),
		},
		{
			name:    "subscriptions",
			backend: subscriptions.New(env, db, m),
		},
	}

	return b, nil
}

func (b *backendManager) Run(ctx context.Context, stop <-chan struct{}) {
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
		for _, backend := range b.registry {
			b.mu.Lock()
			for atomic.LoadInt32(&backend.workers) >= maxWorkers && !b.stopping.Load().(bool) {
				b.cond.Wait()
			}
			b.mu.Unlock()
		}

		if b.stopping.Load().(bool) {
			break
		}

		backendDidWork := make(map[string]bool, len(b.registry))
		for _, backendInstance := range b.registry {
			var err error
			backendDidWork[backendInstance.name], err = b.try(ctx, backendInstance)
			if err != nil {
				b.baseLog.Error(err)
			}
		}

		for _, didWork := range backendDidWork {
			if !didWork {
				<-t.C
			}
		}
	}
}

// try tries to dequeue an object for work, and works it on a
// new goroutine.  It returns a boolean to the caller indicating whether it
// succeeded in dequeuing anything - if this is false, the caller should sleep
// before calling again
func (b *backendManager) try(ctx context.Context, instance instance) (bool, error) {
	log := b.baseLog.WithField("backend", instance.name)
	atomic.AddInt32(&instance.workers, 1)
	b.m.EmitGauge(fmt.Sprintf("backend.%s.workers.count", instance.name), int64(atomic.LoadInt32(&instance.workers)), nil)

	docRaw, err := instance.backend.Dequeue()(ctx)
	if err != nil || docRaw == nil {
		return false, err
	}

	go func() {
		defer recover.Panic(log)

		t := time.Now()

		defer func() {
			atomic.AddInt32(&instance.workers, -1)
			b.m.EmitGauge(fmt.Sprintf("backend.%s.workers.count", instance.name), int64(atomic.LoadInt32(&instance.workers)), nil)
			b.cond.Signal()

			log.WithField("duration", time.Now().Sub(t).Seconds()).Print("done")
		}()

		err = instance.backend.Handle(ctx, b.baseLog, docRaw)
		if err != nil {
			log.Error(err)
		}

	}()

	return true, nil
}
