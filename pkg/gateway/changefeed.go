package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

func (g *gateway) changefeed(ctx context.Context) {
	defer recover.Panic(g.log)

	gwIterator := g.dbGateway.ChangeFeed()

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	g.updateFromIterator(ctx, t, gwIterator)
}

func (g *gateway) updateFromIterator(ctx context.Context, ticker *time.Ticker, gwIterator cosmosdb.GatewayDocumentIterator) {

	for {
		docs, err := gwIterator.Next(ctx, -1)
		successful := true

		for ; docs != nil && err == nil; docs, err = gwIterator.Next(ctx, -1) {
			g.updateGateways(docs.GatewayDocuments)
		}
		if err != nil {
			successful = false
			g.log.Error(err)
		}

		if successful {
			g.lastChangefeed.Store(time.Now())
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}
