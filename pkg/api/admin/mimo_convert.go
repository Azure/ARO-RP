package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type maintenanceManifestConverter struct{}

func (m maintenanceManifestConverter) ToExternal(d *api.MaintenanceManifestDocument) interface{} {
	return &MaintenanceManifest{
		ID: d.ID,

		State:      MaintenanceManifestState(d.MaintenanceManifest.State),
		StatusText: d.MaintenanceManifest.StatusText,

		MaintenanceSetID: d.MaintenanceManifest.MaintenanceSetID,
		Priority:         d.MaintenanceManifest.Priority,

		RunAfter:  d.MaintenanceManifest.RunAfter,
		RunBefore: d.MaintenanceManifest.RunBefore,
	}
}

func (m maintenanceManifestConverter) ToExternalList(docs []*api.MaintenanceManifestDocument, nextLink string) interface{} {
	l := &MaintenanceManifestList{
		MaintenanceManifests: make([]*MaintenanceManifest, 0, len(docs)),
		NextLink:             nextLink,
	}

	for _, doc := range docs {
		l.MaintenanceManifests = append(l.MaintenanceManifests, m.ToExternal(doc).(*MaintenanceManifest))
	}

	return l
}

func (m maintenanceManifestConverter) ToInternal(_i interface{}, out *api.MaintenanceManifestDocument) {

	i := _i.(*MaintenanceManifest)

	out.ID = i.ID
	out.MaintenanceManifest.MaintenanceSetID = i.MaintenanceSetID
	out.MaintenanceManifest.Priority = i.Priority
	out.MaintenanceManifest.RunAfter = i.RunAfter
	out.MaintenanceManifest.RunBefore = i.RunBefore
	out.MaintenanceManifest.State = api.MaintenanceManifestState(i.State)
	out.MaintenanceManifest.StatusText = i.StatusText
}
