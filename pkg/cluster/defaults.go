package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
)

// ensureDefaults will ensure cluster documents has all default values
// for new api versions
func (m *manager) ensureDefaults(ctx context.Context) error {
	return patchClusterDoc(ctx, m, func(doc *api.OpenShiftClusterDocument) error {
		api.SetDefaults(doc)
		return nil
	})
}

// ensureBYONsg updates the cluster docs to disable operators that might interfere
// with BYO NSG feature
func (m *manager) ensureBYONsg(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG {
		return patchClusterDoc(ctx, m, func(doc *api.OpenShiftClusterDocument) error {
			flags := doc.OpenShiftCluster.Properties.OperatorFlags
			flags["aro.azuresubnets.nsg.managed"] = "false"
			return nil
		})
	}
	return nil
}

func patchClusterDoc(ctx context.Context, m *manager, mut database.OpenShiftClusterDocumentMutator) error {
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, mut)
	if err != nil {
		m.log.Print(err)
	}
	return err
}
