package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func ExampleClusterManagerConfigurationDocument() *ClusterManagerConfigurationDocument {
	return &ClusterManagerConfigurationDocument{
		ID:           "00000000-0000-0000-0000-000000000000",
		Key:          "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/syncSets/mySyncSet",
		ResourceID:   "",
		PartitionKey: "",
		ClusterManagerConfiguration: &ClusterManagerConfiguration{
			ID:                "/subscriptions/subscriptionId/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/resourceName/syncSets/mySyncSet",
			ClusterResourceId: "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
			Kind:              SyncSetKind,
			Resources:         []byte("eyJzcGVjIjp7IkNvbmZpZ01hcCI6eyJUeXBlTWV0YSI6eyJBUElWZXJzaW9uIjoidjEiLCJLaW5kIjoiQ29uZmlnTWFwIn19fX0="),
		},
		CorrelationData: &CorrelationData{},
	}
}
