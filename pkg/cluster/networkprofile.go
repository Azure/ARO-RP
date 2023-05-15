package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/feature"
)

// populateMTUSize ensures that every new cluster object has the MTUSize field defined
func (m *manager) populateMTUSize(ctx context.Context) error {
	// Get appropriate MTU size
	mtuSize := api.MTU1500
	subProperties := m.subscriptionDoc.Subscription.Properties
	if feature.IsRegisteredForFeature(subProperties, api.FeatureFlagMTU3900) {
		mtuSize = api.MTU3900
	}

	// Patch the cluster object with correct MTU size
	return patchMTUSize(m, ctx, mtuSize)
}

// ensureMTUSize ensures that an existing cluster object has the MTUSize field defined
func (m *manager) ensureMTUSize(ctx context.Context) error {
	var err error
	// Cluster needs MTUSize field patched
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.MTUSize == 0 {
		// Get appropriate MTU size
		mtuSize := api.MTU3900

		// If a single MachineConfig is present we know the cluster has a custom MTU
		_, err = m.mcocli.MachineconfigurationV1().MachineConfigs().Get(ctx, "99-master-mtu", metav1.GetOptions{})
		if err != nil {
			// If we can't find a MachineConfig this cluster never had a custom MTU on install, set to default
			if kerrors.IsNotFound(err) {
				mtuSize = api.MTU1500
			} else {
				return err
			}
		}

		// Patch the cluster object with correct MTU size
		err = patchMTUSize(m, ctx, mtuSize)
	}
	return err
}

func patchMTUSize(m *manager, ctx context.Context, mtuSize api.MTUSize) error {
	// Patch the cluster object with correct MTU size
	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.NetworkProfile.MTUSize = mtuSize
		return nil
	})
	return err
}
