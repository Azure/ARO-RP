package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
)

// ensureDefaults will ensure cluster documents has all default values
// for new api versions
func (m *manager) ensureDefaults(ctx context.Context) error {
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		api.SetDefaults(doc)
		return nil
	})
	if err != nil {
		m.log.Print(err)
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeDeploymentFailed,
			"Cluster Default Values",
			"Error validating the cluster default values.",
		)
	}
	return nil
}
