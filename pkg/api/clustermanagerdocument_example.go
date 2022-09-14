package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// ExampleClusterManagerConfigurationDocumentSyncSet returns a ClusterManagerConfigurationDocument
// with an example syncset payload model. The resources field comes from the ./hack/ocm folder.
func ExampleClusterManagerConfigurationDocumentSyncSet() *ClusterManagerConfigurationDocument {
	return &ClusterManagerConfigurationDocument{
		ID:           "00000000-0000-0000-0000-000000000000",
		Key:          "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/syncSets/mySyncSet",
		ResourceID:   "",
		PartitionKey: "",
		SyncSet: &SyncSet{
			Name: "mySyncSet",
			ID:   "/subscriptions/subscriptionId/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/resourceName/syncSets/mySyncSet",
			Type: "Microsoft.RedHatOpenShift/OpenShiftClusters/SyncSets",
			Properties: SyncSetProperties{
				Resources: "eyAKICAiYXBpVmVyc2lvbiI6ICJoaXZlLm9wZW5zaGlmdC5pby92MSIsCiAgImtpbmQiOiAiU3luY1NldCIsCiAgIm1ldGFkYXRhIjogewogICAgIm5hbWUiOiAic2FtcGxlIiwKICAgICJuYW1lc3BhY2UiOiAiYXJvLWY2MGFlOGEyLWJjYTEtNDk4Ny05MDU2LWYyZjZhMTgzN2NhYSIKICB9LAogICJzcGVjIjogewogICAgImNsdXN0ZXJEZXBsb3ltZW50UmVmcyI6IFtdLAogICAgInJlc291cmNlcyI6IFsKICAgICAgewogICAgICAgICJhcGlWZXJzaW9uIjogInYxIiwKICAgICAgICAia2luZCI6ICJDb25maWdNYXAiLAogICAgICAgICJtZXRhZGF0YSI6IHsKICAgICAgICAgICJuYW1lIjogIm15Y29uZmlnbWFwIgogICAgICAgIH0KICAgICAgfQogICAgXQogIH0KfQo=",
			},
		},
		CorrelationData: &CorrelationData{},
	}
}

func ExampleClusterManagerConfigurationDocumentMachinePool() *ClusterManagerConfigurationDocument {
	return &ClusterManagerConfigurationDocument{
		ID:           "00000000-0000-0000-0000-000000000000",
		Key:          "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/machinepools/myMachinePool",
		ResourceID:   "",
		PartitionKey: "",
		MachinePool: &MachinePool{
			Name: "myMachinePool",
			ID:   "/subscriptions/subscriptionId/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/resourceName/machinePools/myMachinePool",
			Type: "Microsoft.RedHatOpenShift/OpenShiftClusters/MachinePools",
			Properties: MachinePoolProperties{
				Resources: "ewogICAgImFwaVZlcnNpb24iOiAiaGl2ZS5vcGVuc2hpZnQuaW8vdjEiLAogICAgImtpbmQiOiAiTWFjaGluZVBvb2wiLAogICAgIm1ldGFkYXRhIjogewogICAgICAgICJuYW1lIjogInRlc3QtY2x1c3Rlci13b3JrZXIiLAogICAgICAgICJuYW1lc3BhY2UiOiAiYXJvLWY2MGFlOGEyLWJjYTEtNDk4Ny05MDU2LVhYWFhYWFhYWFhYWCIKICAgIH0sCiAgICAic3BlYyI6IHsKICAgICAgICAiY2x1c3RlckRlcGxveW1lbnRSZWYiOiB7CiAgICAgICAgICAgICJuYW1lIjogInRlc3QtY2x1c3RlciIKICAgICAgICB9LAogICAgICAgICJuYW1lIjogIndvcmtlciIsCiAgICAgICAgInBsYXRmb3JtIjogewogICAgICAgICAgICAiYXdzIjogewogICAgICAgICAgICAgICAgInJvb3RWb2x1bWUiOiB7CiAgICAgICAgICAgICAgICAgICAgImlvcHMiOiAwLAogICAgICAgICAgICAgICAgICAgICJzaXplIjogMzAwLAogICAgICAgICAgICAgICAgICAgICJ0eXBlIjogImdwMyIKICAgICAgICAgICAgICAgIH0sCiAgICAgICAgICAgICAgICAidHlwZSI6ICJtNS54bGFyZ2UiLAogICAgICAgICAgICAgICAgInpvbmVzIjogWwogICAgICAgICAgICAgICAgICAgICJ1cy1lYXN0LTFhIgogICAgICAgICAgICAgICAgXQogICAgICAgICAgICB9CiAgICAgICAgfSwKICAgICAgICAicmVwbGljYXMiOiAyCiAgICB9LAogICAgInN0YXR1cyI6IHsKICAgICAgICAiY29uZGl0aW9ucyI6IFsKICAgICAgICBdCiAgICB9Cn0K",
			},
		},
		CorrelationData: &CorrelationData{},
	}
}

func ExampleClusterManagerConfigurationDocumentSyncIdentityProvider() *ClusterManagerConfigurationDocument {
	return &ClusterManagerConfigurationDocument{
		ID:           "00000000-0000-0000-0000-000000000000",
		Key:          "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/syncidentityprovider/mySyncIdentityProvider",
		ResourceID:   "",
		PartitionKey: "",
		SyncIdentityProvider: &SyncIdentityProvider{
			Name: "mySyncIdentityProvider",
			Type: "Microsoft.RedHatOpenShift/OpenShiftClusters/SyncIdentityProviders",
			ID:   "/subscriptions/subscriptionId/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/resourceName/syncidentityprovider/mySyncIdentityProvider",
			Properties: SyncIdentityProviderProperties{
				Resources: "ewogICAgImFwaVZlcnNpb24iOiAiaGl2ZS5vcGVuc2hpZnQuaW8vdjEiLAogICAgImtpbmQiOiAiU3luY0lkZW50aXR5UHJvdmlkZXIiLAogICAgIm1ldGFkYXRhIjogewogICAgICAgICJuYW1lIjogInRlc3QtY2x1c3RlciIsCiAgICAgICAgIm5hbWVzcGFjZSI6ICJhcm8tZjYwYWU4YTItYmNhMS00OTg3LTkwNTYtWFhYWFhYWFhYWFhYIgogICAgfSwKICAgICJzcGVjIjogewogICAgICAgICJjbHVzdGVyRGVwbG95bWVudFJlZnMiOiBbCiAgICAgICAgICAgIHsKICAgICAgICAgICAgICAgICJuYW1lIjogInRlc3QtY2x1c3RlciIKICAgICAgICAgICAgfQogICAgICAgIF0sCiAgICAgICAgImlkZW50aXR5UHJvdmlkZXJzIjogWwogICAgICAgICAgICB7CiAgICAgICAgICAgICAgICAiaHRwYXNzd2QiOiB7CiAgICAgICAgICAgICAgICAgICAgImZpbGVEYXRhIjogewogICAgICAgICAgICAgICAgICAgICAgICAibmFtZSI6ICJodHBhc3N3ZC1zZWNyZXQiCiAgICAgICAgICAgICAgICAgICAgfQogICAgICAgICAgICAgICAgfSwKICAgICAgICAgICAgICAgICJtYXBwaW5nTWV0aG9kIjogImNsYWltIiwKICAgICAgICAgICAgICAgICJuYW1lIjogIkhUUGFzc3dkIiwKICAgICAgICAgICAgICAgICJ0eXBlIjogIkhUUGFzc3dkIgogICAgICAgICAgICB9CiAgICAgICAgXQogICAgfSwKICAgICJzdGF0dXMiOiB7fQp9Cg==",
			},
		},
		CorrelationData: &CorrelationData{},
	}
}

func ExampleClusterManagerConfigurationDocumentSecret() *ClusterManagerConfigurationDocument {
	return &ClusterManagerConfigurationDocument{
		ID:           "00000000-0000-0000-0000-000000000000",
		Key:          "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/machinepools/mySyncIdentityProvider",
		ResourceID:   "",
		PartitionKey: "",
		Secret: &Secret{
			Name: "mySecret",
			Type: "Microsoft.RedHatOpenShift/OpenShiftClusters/Secrets",
			ID:   "/subscriptions/subscriptionId/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/resourceName/secret/mySecret",
			Properties: SecretProperties{
				SecretResources: SecureString("YXBpVmVyc2lvbjogdjEKZGF0YToKICB1c2VybmFtZTogWVdSdGFXND0KICBwYXNzd29yZDogTVdZeVpERmxNbVUyTjJSbQpraW5kOiBTZWNyZXQKbWV0YWRhdGE6CiAgYW5ub3RhdGlvbnM6CiAgICBrdWJlY3RsLmt1YmVybmV0ZXMuaW8vbGFzdC1hcHBsaWVkLWNvbmZpZ3VyYXRpb246IHsgLi4uIH0KICBjcmVhdGlvblRpbWVzdGFtcDogMjAyMC0wMS0yMlQxODo0MTo1NloKICBuYW1lOiBteXNlY3JldAogIG5hbWVzcGFjZTogZGVmYXVsdAogIHJlc291cmNlVmVyc2lvbjogMTY0NjE5CiAgdWlkOiBjZmVlMDJkNi1jMTM3LTExZTUtOGQ3My00MjAxMGFmMDAwMDIKdHlwZTogT3BhcXVlCg=="),
			},
		},
		CorrelationData: &CorrelationData{},
	}
}
