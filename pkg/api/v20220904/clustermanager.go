package v20220904

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	hivev1 "github.com/openshift/hive/apis/hive/v1"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// ClusterManagerConfiguration represents the configuration from OpenShift Cluster Manager (OCM)
type ClusterManagerConfiguration struct {
	ID       string `json:"id,omitempty"`
	Deleting bool   `json:"deleting,omitempty"` // https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes

	SyncSets map[string]hivev1.SyncSet `json:"syncSets,omitempty"`
}
