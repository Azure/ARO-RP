package changefeed

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type ChangefeedDocumentList[E any] interface {
	Docs() []E
	GetCount() int
}

type Changefeed[D any] interface {
	Next(context.Context, int) (D, error)
}

type ChangefeedResponder[F any] interface {
	OnDoc(F)
	OnAllPendingProcessed()
	Lock()
	Unlock()
}

func NewChangefeed[F any, X ChangefeedDocumentList[F]](
	ctx context.Context,
	log *logrus.Entry,
	iterator Changefeed[X],
	changefeedInterval time.Duration,
	changefeedBatchSize int,
	responder ChangefeedResponder[F],
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
				log.Error(err)
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
