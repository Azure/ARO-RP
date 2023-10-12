package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/recover"
	"github.com/sirupsen/logrus"
)

const (
	maxWorkers      = 100
	maxDequeueCount = 5
)

type Runnable interface {
	Run(context.Context, <-chan struct{}, chan<- struct{})
	Workers() *atomic.Int64
	Cond() *sync.Cond
	Wait()
}

type WorkTask func(context.Context, *sync.Cond) (bool, error)

type workerQueue struct {
	baseLog *logrus.Entry

	task WorkTask

	disableReadinessDelay bool
	mu                    sync.Mutex
	cond                  *sync.Cond
	workers               *atomic.Int64
	stopping              *atomic.Bool
}

func NewWorkerQueue(ctx context.Context, log *logrus.Entry, _env env.Interface, task WorkTask) Runnable {
	waitForWorkerCompletion := false
	if !_env.FeatureIsSet(env.FeatureDisableReadinessDelay) {
		waitForWorkerCompletion = true
	}

	q := &workerQueue{
		baseLog: log,

		workers:               &atomic.Int64{},
		stopping:              &atomic.Bool{},
		task:                  task,
		disableReadinessDelay: waitForWorkerCompletion,
	}
	q.cond = sync.NewCond(&q.mu)
	q.stopping.Store(false)
	return q
}

func (b *workerQueue) Run(ctx context.Context, stop <-chan struct{}, done chan<- struct{}) {
	defer recover.Panic(b.baseLog)

	t := time.NewTicker(10 * time.Second)
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
		for b.workers.Load() >= maxWorkers && !b.stopping.Load() {
			b.cond.Wait()
		}
		b.mu.Unlock()

		if b.stopping.Load() {
			break
		}

		workDone, err := b.task(ctx, b.cond)
		if err != nil {
			b.baseLog.Error(err)
		}

		if !workDone {
			<-t.C
		}
	}

	if b.disableReadinessDelay {
		b.Wait()
	}
	b.baseLog.Print("exiting")
	close(done)
}

func (b *workerQueue) Wait() {
	b.mu.Lock()
	for b.workers.Load() > 0 {
		b.cond.Wait()
	}
	b.mu.Unlock()
}

func (b *workerQueue) Workers() *atomic.Int64 {
	return b.workers
}

func (b *workerQueue) Cond() *sync.Cond {
	return b.cond
}
