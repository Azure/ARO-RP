package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Gateway represents a Gateway entry
type Gateway struct {
	MissingFields

	ID       string `json:"id,omitempty"`
	Deleting bool   `json:"deleting,omitempty"` // https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes

	StorageSuffix                   string `json:"storageSuffix,omitempty"`
	ImageRegistryStorageAccountName string `json:"imageRegistryStorageAccountName,omitempty"`
}
