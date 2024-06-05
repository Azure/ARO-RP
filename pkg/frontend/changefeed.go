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

func (f *frontend) changefeedOcpVersions(ctx context.Context) {
	defer recover.Panic(f.baseLog)

	// f.dbOpenShiftVersions will be nil when running unit tests. Return here to avoid nil pointer panic
	if f.dbOpenShiftVersions == nil {
		return
	}

	ocpVersionsIterator := f.dbOpenShiftVersions.ChangeFeed()

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	f.updateFromIteratorOcpVersions(ctx, t, ocpVersionsIterator)
}

func (f *frontend) changefeedRoleSets(ctx context.Context) {
	defer recover.Panic(f.baseLog)

	if f.dbPlatformWorkloadIdentityRoleSets == nil {
		return
	}

	roleSetsIterator := f.dbPlatformWorkloadIdentityRoleSets.ChangeFeed()

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	f.updateFromIteratorRoleSets(ctx, t, roleSetsIterator)
}

func (f *frontend) updateFromIteratorOcpVersions(ctx context.Context, ticker *time.Ticker, frontendIterator cosmosdb.OpenShiftVersionDocumentIterator) {
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
			f.lastOcpVersionsChangefeed.Store(time.Now())
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
	f.ocpVersionsMu.Lock()
	defer f.ocpVersionsMu.Unlock()

	for _, doc := range docs {
		if doc.OpenShiftVersion.Deleting || !doc.OpenShiftVersion.Properties.Enabled {
			// https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes
			delete(f.enabledOcpVersions, doc.OpenShiftVersion.Properties.Version)
		} else {
			f.enabledOcpVersions[doc.OpenShiftVersion.Properties.Version] = doc.OpenShiftVersion
			if doc.OpenShiftVersion.Properties.Default {
				f.defaultOcpVersion = doc.OpenShiftVersion.Properties.Version
			}
		}
	}
}

func (f *frontend) updateFromIteratorRoleSets(ctx context.Context, ticker *time.Ticker, frontendIterator cosmosdb.PlatformWorkloadIdentityRoleSetDocumentIterator) {
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

			f.updatePlatformWorkloadIdentityRoleSets(docs.PlatformWorkloadIdentityRoleSetDocuments)
		}

		if successful {
			f.lastPlatformWorkloadIdentityRoleSetsChangefeed.Store(time.Now())
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (f *frontend) updatePlatformWorkloadIdentityRoleSets(docs []*api.PlatformWorkloadIdentityRoleSetDocument) {
	f.platformWorkloadIdentityRoleSetsMu.Lock()
	defer f.platformWorkloadIdentityRoleSetsMu.Unlock()

	for _, doc := range docs {
		if doc.PlatformWorkloadIdentityRoleSet.Deleting {
			// https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes
			delete(f.availablePlatformWorkloadIdentityRoleSets, doc.PlatformWorkloadIdentityRoleSet.Properties.OpenShiftVersion)
		} else {
			f.availablePlatformWorkloadIdentityRoleSets[doc.PlatformWorkloadIdentityRoleSet.Properties.OpenShiftVersion] = doc.PlatformWorkloadIdentityRoleSet
		}
	}
}
