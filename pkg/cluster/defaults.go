package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

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
		return err
	}
	return nil
}

// ensurePreconfiguredNSG updates the cluster docs to disable operators that might interfere
// with customers bringing their own NSG feature
func (m *manager) ensurePreconfiguredNSG(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG == api.PreconfiguredNSGEnabled {
		var err error
		m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			flags := doc.OpenShiftCluster.Properties.OperatorFlags
			flags["aro.azuresubnets.nsg.managed"] = "false"
			return nil
		})
		if err != nil {
			m.log.Error(err)
			return err
		}
	}
	return nil
}
