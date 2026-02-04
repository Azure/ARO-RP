package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/operator"
)

// ensureDefaults will ensure cluster documents has all default values
// for new api versions
func (m *manager) ensureDefaults(ctx context.Context) error {
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		api.SetDefaults(doc, operator.DefaultOperatorFlags)
		// SetDNSDefaults will set DNS type based on cluster version (4.21+ uses CustomDNS)
		api.SetDNSDefaults(doc)
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
			flags[operator.AzureSubnetsNsgManaged] = operator.FlagFalse
			return nil
		})
		if err != nil {
			m.log.Error(err)
			return err
		}
	}
	return nil
}
