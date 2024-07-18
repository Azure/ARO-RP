package admin

import "github.com/Azure/ARO-RP/pkg/api"

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
