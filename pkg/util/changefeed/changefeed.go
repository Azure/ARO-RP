package changefeed

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

const (
	// MetricStaleness is the number of seconds since the changefeed last
	// completed a poll without error. It rises without bound while a feed is
	// stalled, and is the signal to alert on.
	MetricStaleness = "changefeed.poll.staleness"

	// MetricConsecutiveFailures is the number of polls which have failed in
	// succession. It tells a stalled feed apart from a quiet one.
	MetricConsecutiveFailures = "changefeed.poll.failures"

	// MetricPagesSkipped is the number of pages the feed has given up on and
	// advanced past, whose contents were therefore never delivered. It only
	// rises, so that a skip stays visible after the feed recovers and the
	// failure count resets.
	MetricPagesSkipped = "changefeed.pages.skipped"
)

// A skipReporter is an iterator which can say how many pages it has given up
// on. Iterators which cannot skip do not implement it, and report nothing.
type skipReporter interface {
	PagesSkipped() int
}

// Generic interface of a consumer that NewChangefeed will call with documents,
// completed pages, etc.
type ChangefeedConsumer[F any] interface {
	// OnDoc is called with each document returned from the list from Next()
	OnDoc(F)
	// OnAllPendingProcessed is when no more pages are returned from Next() with
	// whether any documents were retrieved during this timer iteration
	OnAllPendingProcessed(bool)
	// Lock is called before a page is processed
	Lock()
	// Unlock is called after a page is processed
	Unlock()
}

// RunChangefeed polls iterator until stop is closed, passing each document to
// responder.
//
// name identifies the feed in logs and metrics; it is conventionally the
// document type, as for changefeed.caches.size.
func RunChangefeed[F any, X api.DocumentList[F]](
	ctx context.Context,
	log *logrus.Entry,
	m metrics.Emitter,
	name string,
	iterator database.DocumentIterator[F, X],
	changefeedInterval time.Duration,
	changefeedBatchSize int,
	responder ChangefeedConsumer[F],
	stop <-chan struct{},
) {
	log = log.WithField("changefeed", name)

	defer recover.Panic(log)

	dimensions := map[string]string{"name": name}

	lastSuccessfulPoll := time.Now()
	consecutiveFailures := 0

	t := time.NewTicker(changefeedInterval)
	defer t.Stop()

	for {
		documentsRetrieved := false
		successful := true
		for {
			docs, err := iterator.Next(ctx, changefeedBatchSize)
			if err != nil {
				successful = false
				log.Errorf("while calling iterator.Next(): %s", err.Error())
				break
			}
			if docs.GetCount() == 0 {
				break
			}

			log.Debugf("changefeed page was %d docs", docs.GetCount())

			responder.Lock()
			for _, doc := range docs.Docs() {
				responder.OnDoc(doc)
			}
			responder.Unlock()
			documentsRetrieved = true
		}

		if successful {
			lastSuccessfulPoll = time.Now()
			consecutiveFailures = 0
			responder.OnAllPendingProcessed(documentsRetrieved)
		} else {
			consecutiveFailures++
		}

		// Emitted on every poll, successful or not. A feed which has stopped
		// advancing is otherwise indistinguishable from one with nothing to
		// report, which is how a month-long stall went unnoticed.
		m.EmitGauge(MetricStaleness, int64(time.Since(lastSuccessfulPoll).Seconds()), dimensions)
		m.EmitGauge(MetricConsecutiveFailures, int64(consecutiveFailures), dimensions)

		if s, ok := iterator.(skipReporter); ok {
			m.EmitGauge(MetricPagesSkipped, int64(s.PagesSkipped()), dimensions)
		}

		select {
		case <-t.C:
		case <-stop:
			return
		}
	}
}
