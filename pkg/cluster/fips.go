package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/feature"
)

func (m *manager) generateFIPSMode(ctx context.Context) error {
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		if feature.IsRegisteredForFeature(m.subscriptionDoc.Subscription.Properties, api.FeatureFlagFIPS) {
			doc.OpenShiftCluster.Properties.ClusterProfile.FipsValidatedModules = api.FipsValidatedModulesEnabled
		} else {
			doc.OpenShiftCluster.Properties.ClusterProfile.FipsValidatedModules = api.FipsValidatedModulesDisabled
		}
		return nil
	})
	return err
}
