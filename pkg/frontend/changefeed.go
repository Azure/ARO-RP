package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

func (f *frontend) changefeed(ctx context.Context) {
	defer recover.Panic(f.baseLog)

	frontendIterator := f.dbOpenShiftVersions.ChangeFeed()

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	f.updateFromIterator(ctx, t, frontendIterator)
}

func (f *frontend) updateFromIterator(ctx context.Context, ticker *time.Ticker, frontendIterator cosmosdb.OpenShiftVersionDocumentIterator) {
	for {
		successful := true

		for {
			docs, err := frontendIterator.Next(ctx, -1)
			if err != nil {
				successful = false
				f.baseLog.Error(err)
				break
			}
			if docs == nil {
				break
			}

			f.updateOcpVersions(docs.OpenShiftVersionDocuments)
		}

		if successful {
			f.lastChangefeed.Store(time.Now())
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

// updateOcpVersions adds enabled versions to the frontend cache
func (f *frontend) updateOcpVersions(docs []*api.OpenShiftVersionDocument) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, doc := range docs {
		if doc.OpenShiftVersion.Deleting || !doc.OpenShiftVersion.Properties.Enabled {
			// https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes
			delete(f.enabledOcpVersions, doc.OpenShiftVersion.Properties.Version)
		} else {
			f.enabledOcpVersions[doc.OpenShiftVersion.Properties.Version] = doc.OpenShiftVersion
		}
	}
}
