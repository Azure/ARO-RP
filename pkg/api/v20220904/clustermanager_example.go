package v20220904

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "github.com/Azure/ARO-RP/pkg/api"

func exampleClusterManagerConfiguration() *ClusterManagerConfiguration {
	doc := api.ExampleClusterManagerConfigurationDocument()
	ext, _ := (&clusterManagerConverter{}).ToExternal(doc.ClusterManagerConfiguration) // swallow err
	return ext.(*ClusterManagerConfiguration)
}
