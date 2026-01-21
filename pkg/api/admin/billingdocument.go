package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// BillingDocument represents a billing document.
type BillingDocument struct {
	// The ID for the resource.
	ID string `json:"id,omitempty"`

	Key                       string `json:"key,omitempty"`
	ClusterResourceGroupIDKey string `json:"clusterResourceGroupIDKey,omitempty"`
	InfraID                   string `json:"infraId,omitempty"`

	Billing *Billing `json:"billing,omitempty"`
}

// Billing represents a Billing entry
type Billing struct {
	CreationTime    int `json:"creationTime,omitempty"`
	DeletionTime    int `json:"deletionTime,omitempty"`
	LastBillingTime int `json:"lastBillingTime,omitempty"`

	Location string `json:"location,omitempty"`
	TenantID string `json:"tenantID,omitempty"`
}

// BillingDocumentList represents a list of BillingDocuments.
type BillingDocumentList struct {
	// The list of BillingDocuments.
	BillingDocuments []*BillingDocument `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}
