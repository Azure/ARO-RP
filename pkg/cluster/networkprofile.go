package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (m *manager) reconcileSoftwareDefinedNetwork(ctx context.Context) error {
	// Clusters that are using SDN will need to migrate to OVN before upgrading to OpenShift 4.17.z
	// This checks any cluster that is still marked as using SDN in its cluster doc, and updates it
	// if the on-cluster networkType has been updated to use OVN
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.SoftwareDefinedNetwork == api.SoftwareDefinedNetworkOpenShiftSDN {
		network, err := m.configcli.ConfigV1().Networks().Get(ctx, "cluster", metav1.GetOptions{})
		if err != nil {
			return err
		}
		if network.Spec.NetworkType == string(api.SoftwareDefinedNetworkOVNKubernetes) {
			m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
				doc.OpenShiftCluster.Properties.NetworkProfile.SoftwareDefinedNetwork = api.SoftwareDefinedNetworkOVNKubernetes
				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
