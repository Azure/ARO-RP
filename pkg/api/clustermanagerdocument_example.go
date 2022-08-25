package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const ssPayload string = `
{
  "apiVersion": "hive.openshift.io/v1",
  "kind": "SyncSet",
  "metadata": {
    "name": "sample",
    "namespace": "aro-f60ae8a2-bca1-4987-9056-f2f6a1837caa"
  },
  "spec": {
    "clusterDeploymentRefs": [],
    "resources": [
      {
        "apiVersion": "v1",
        "kind": "ConfigMap",
        "metadata": {
          "name": "myconfigmap"
        }
      }
    ]
  }
}
`

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
        "platform": {
            "aws": {
                "rootVolume": {
                    "iops": 0,
                    "size": 300,
                    "type": "gp3"
                },
                "type": "m5.xlarge",
                "zones": [
                    "us-east-1a"
                ]
            }
        },
        "replicas": 2
    },
    "status": {
        "conditions": [
        ]
    }
}
`

func ExampleClusterManagerConfigurationDocumentSyncSet() *ClusterManagerConfigurationDocument {
	return &ClusterManagerConfigurationDocument{
		ID:           "00000000-0000-0000-0000-000000000000",
		Key:          "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/syncSets/mySyncSet",
		ResourceID:   "",
		PartitionKey: "",
		ClusterManagerConfiguration: &ClusterManagerConfiguration{
			Name:              "mySyncSet",
			ID:                "/subscriptions/subscriptionId/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/resourceName/syncSets/mySyncSet",
			ClusterResourceId: "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
			Properties: ClusterManagerConfigurationProperties{
				Resources: []byte(ssPayload),
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
		ClusterManagerConfiguration: &ClusterManagerConfiguration{
			Name:              "myMachinePool",
			ID:                "/subscriptions/subscriptionId/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/resourceName/machinePools/myMachinePool",
			ClusterResourceId: "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
			Properties: ClusterManagerConfigurationProperties{
				Resources: []byte(mpPayload),
			},
		},
		CorrelationData: &CorrelationData{},
	}
}
