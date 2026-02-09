package changefeed

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

// Generic interface of a consumer that NewChangefeed will call with documents,
// completed pages, etc.
type ChangefeedConsumer[F any] interface {
	// OnDoc is called with each document returned from the list from Next()
	OnDoc(F)
	// OnAllPendingProcessed is when no more pages are returned from Next()
	OnAllPendingProcessed()
	// Lock is called before a page is processed
	Lock()
	// Unlock is called after a page is processed
	Unlock()
}

func NewChangefeed[F any, X api.DocumentList[F]](
	ctx context.Context,
	log *logrus.Entry,
	iterator database.DocumentIterator[F, X],
	changefeedInterval time.Duration,
	changefeedBatchSize int,
	responder ChangefeedConsumer[F],
	stop <-chan struct{},
) {
	defer recover.Panic(log)

	t := time.NewTicker(changefeedInterval)
	defer t.Stop()

	for {
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
		}

		if successful {
			responder.OnAllPendingProcessed()
		}

		select {
		case <-t.C:
		case <-stop:
			return
		}
	}
}
