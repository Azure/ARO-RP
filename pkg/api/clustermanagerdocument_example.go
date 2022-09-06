package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const mpPayload string = `
{
    "apiVersion": "hive.openshift.io/v1",
    "kind": "MachinePool",
    "metadata": {
        "creationTimestamp": "2022-08-16T14:17:10Z",
        "generation": 1,
        "labels": {
            "api.openshift.com/id": "1u4lhakk4ar41bi3vgn0b7v9hk93dg4m"
        },
        "name": "oadler-full-worker",
        "namespace": "uhc-staging-1u4lhakk4ar41bi3vgn0b7v9hk93dg4m",
        "resourceVersion": "1205855122",
        "uid": "28a4de99-dc5f-4a9a-9f50-94a7dd47c712"
    },
    "spec": {
        "clusterDeploymentRef": {
            "name": "oadler-full"
        },
        "name": "worker",
        "platform": {}
        },
        "replicas": 2
    },
    "status": {
        "conditions": [
        ]
    }
}
`

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
			Properties: SyncSetProperties{
				ClusterResourceId: "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
				Resources:         "eyAKICAiYXBpVmVyc2lvbiI6ICJoaXZlLm9wZW5zaGlmdC5pby92MSIsCiAgImtpbmQiOiAiU3luY1NldCIsCiAgIm1ldGFkYXRhIjogewogICAgIm5hbWUiOiAic2FtcGxlIiwKICAgICJuYW1lc3BhY2UiOiAiYXJvLWY2MGFlOGEyLWJjYTEtNDk4Ny05MDU2LWYyZjZhMTgzN2NhYSIKICB9LAogICJzcGVjIjogewogICAgImNsdXN0ZXJEZXBsb3ltZW50UmVmcyI6IFtdLAogICAgInJlc291cmNlcyI6IFsKICAgICAgewogICAgICAgICJhcGlWZXJzaW9uIjogInYxIiwKICAgICAgICAia2luZCI6ICJDb25maWdNYXAiLAogICAgICAgICJtZXRhZGF0YSI6IHsKICAgICAgICAgICJuYW1lIjogIm15Y29uZmlnbWFwIgogICAgICAgIH0KICAgICAgfQogICAgXQogIH0KfQo=",
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
			Properties: MachinePoolProperties{
				ClusterResourceId: "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
				Resources:         []byte(mpPayload),
			},
		},
		CorrelationData: &CorrelationData{},
	}
}
