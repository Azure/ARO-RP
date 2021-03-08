package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/Azure/ARO-RP/pkg/util/recover"
)

func (g *gateway) changefeed(ctx context.Context) {
	defer recover.Panic(g.log)

	gwIterator := g.dbGateway.ChangeFeed()

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	for {
		successful := true

		for {
			docs, err := gwIterator.Next(ctx, -1)
			if err != nil {
				successful = false
				g.log.Error(err)
				break
			}
			if docs == nil {
				break
			}

			g.mu.Lock()

			for _, doc := range docs.GatewayDocuments {
				if doc.Gateway.Deleting {
					// https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes
					delete(g.gateways, doc.ID)
				} else {
					g.gateways[doc.ID] = doc.Gateway
				}
			}

			g.mu.Unlock()
		}

		if successful {
			g.lastChangefeed.Store(time.Now())
		}

		select {
		case <-t.C:
		case <-ctx.Done():
			return
		}
	}
}
