package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/puzpuzpuz/xsync/v4"

	"github.com/Azure/ARO-RP/pkg/api"
)

type cacheDoc struct {
	doc  *api.OpenShiftClusterDocument
	stop chan<- struct{}
}

// deleteDoc deletes the given document from mon.docs, signalling the associated
// monitoring goroutine to stop if it exists.
func (c *clusterChangeFeedResponder) deleteDoc(doc *api.OpenShiftClusterDocument) {
	c.docs.Compute(doc.ID, func(oldValue *cacheDoc, loaded bool) (newValue *cacheDoc, op xsync.ComputeOp) {
		if loaded && oldValue.stop != nil {
			close(oldValue.stop)
		}
		return nil, xsync.DeleteOp
	})
}

// upsertDoc inserts or updates the given document into mon.docs, starting an
// associated monitoring goroutine if the document is in a bucket owned by us.
func (c *clusterChangeFeedResponder) upsertDoc(doc *api.OpenShiftClusterDocument) {
	c.bucketMu.RLock()
	defer c.bucketMu.RUnlock()
	c.docs.Compute(doc.ID, func(oldValue *cacheDoc, loaded bool) (newValue *cacheDoc, op xsync.ComputeOp) {
		if loaded {
			oldValue.doc = stripUnusedFields(doc)
			c.fixDoc(oldValue)
			return oldValue, xsync.UpdateOp
		} else {
			newValue = &cacheDoc{doc: stripUnusedFields(doc)}
			c.fixDoc(newValue)
			return newValue, xsync.UpdateOp
		}
	})
}

// stripUnusedFields creates a copy of the document with only the fields
// required for monitoring. This significantly reduces memory usage by
// excluding large fields like kubeconfigs, secrets, and other data not
// needed for cluster monitoring.
//
// Fields retained for monitoring:
// - Document metadata: ID, Key, PartitionKey, Bucket
// - Cluster identity: ID, Name, Location, Type
// - Cluster state: ProvisioningState, FailedProvisioningState, ProvisionedBy, CreatedAt
// - Network config: NetworkProfile (for API server IP and NSG checks)
// - Profiles needed: MasterProfile, WorkerProfiles (for subnet monitoring)
// - API access: APIServerProfile, one kubeconfig (AROServiceKubeconfig preferred)
// - Hive integration: HiveProfile
// - Auth type detection: PlatformWorkloadIdentityProfile, ServicePrincipalProfile (presence only)
func stripUnusedFields(doc *api.OpenShiftClusterDocument) *api.OpenShiftClusterDocument {
	if doc == nil || doc.OpenShiftCluster == nil {
		return doc
	}

	oc := doc.OpenShiftCluster

	// Select the kubeconfig to keep - prefer AROServiceKubeconfig
	var kubeconfigToKeep api.SecureBytes
	if oc.Properties.AROServiceKubeconfig != nil {
		kubeconfigToKeep = oc.Properties.AROServiceKubeconfig
	} else {
		kubeconfigToKeep = oc.Properties.AdminKubeconfig
	}

	// Create stripped worker profiles (only need Name, SubnetID, Count)
	var strippedWorkerProfiles []api.WorkerProfile
	if oc.Properties.WorkerProfiles != nil {
		strippedWorkerProfiles = make([]api.WorkerProfile, len(oc.Properties.WorkerProfiles))
		for i, wp := range oc.Properties.WorkerProfiles {
			strippedWorkerProfiles[i] = api.WorkerProfile{
				Name:     wp.Name,
				SubnetID: wp.SubnetID,
				Count:    wp.Count,
			}
		}
	}

	// Create minimal ServicePrincipalProfile (only need to check presence, not credentials)
	var strippedSPProfile *api.ServicePrincipalProfile
	if oc.Properties.ServicePrincipalProfile != nil {
		strippedSPProfile = &api.ServicePrincipalProfile{
			ClientID: oc.Properties.ServicePrincipalProfile.ClientID,
			// Intentionally omit ClientSecret - not needed for monitoring
		}
	}

	// Create minimal PlatformWorkloadIdentityProfile (only need to check presence)
	var strippedPWIProfile *api.PlatformWorkloadIdentityProfile
	if oc.Properties.PlatformWorkloadIdentityProfile != nil {
		strippedPWIProfile = &api.PlatformWorkloadIdentityProfile{}
	}

	// Build the stripped document
	stripped := &api.OpenShiftClusterDocument{
		ID:           doc.ID,
		Key:          doc.Key,
		PartitionKey: doc.PartitionKey,
		Bucket:       doc.Bucket,
		OpenShiftCluster: &api.OpenShiftCluster{
			ID:       oc.ID,
			Name:     oc.Name,
			Type:     oc.Type,
			Location: oc.Location,
			Properties: api.OpenShiftClusterProperties{
				ProvisioningState:       oc.Properties.ProvisioningState,
				FailedProvisioningState: oc.Properties.FailedProvisioningState,
				ProvisionedBy:           oc.Properties.ProvisionedBy,
				CreatedAt:               oc.Properties.CreatedAt,
				ClusterProfile: api.ClusterProfile{
					Domain:  oc.Properties.ClusterProfile.Domain,
					Version: oc.Properties.ClusterProfile.Version,
					// Intentionally omit PullSecret, BoundServiceAccountSigningKey
				},
				NetworkProfile: api.NetworkProfile{
					PodCIDR:                    oc.Properties.NetworkProfile.PodCIDR,
					ServiceCIDR:                oc.Properties.NetworkProfile.ServiceCIDR,
					APIServerPrivateEndpointIP: oc.Properties.NetworkProfile.APIServerPrivateEndpointIP,
					PreconfiguredNSG:           oc.Properties.NetworkProfile.PreconfiguredNSG,
				},
				MasterProfile: api.MasterProfile{
					SubnetID: oc.Properties.MasterProfile.SubnetID,
				},
				WorkerProfiles: strippedWorkerProfiles,
				APIServerProfile: api.APIServerProfile{
					Visibility: oc.Properties.APIServerProfile.Visibility,
					URL:        oc.Properties.APIServerProfile.URL,
					IP:         oc.Properties.APIServerProfile.IP,
				},
				AROServiceKubeconfig:            kubeconfigToKeep,
				HiveProfile:                     oc.Properties.HiveProfile,
				ServicePrincipalProfile:         strippedSPProfile,
				PlatformWorkloadIdentityProfile: strippedPWIProfile,
				MaintenanceState:                oc.Properties.MaintenanceState,
			},
		},
	}

	return stripped
}

// fixDocs ensures that there is a monitoring goroutine for all documents in all
// buckets owned by us. Caller needs to own r.bucketMu.
func (c *clusterChangeFeedResponder) fixDocs() {
	for _, v := range c.docs.All() {
		c.fixDoc(v)
	}
}

// fixDoc ensures that there is a monitoring goroutine for the given document
// if it is in a bucket owned by us. Caller needs to own r.bucketMu.
func (c *clusterChangeFeedResponder) fixDoc(v *cacheDoc) {
	_, ours := c.buckets[v.doc.Bucket]

	if !ours && v.stop != nil {
		close(v.stop)
		v.stop = nil
	} else if ours && v.stop == nil {
		ch := make(chan struct{})
		v.stop = ch
		go c.newWorker(ch, v.doc.ID)
	}
}
