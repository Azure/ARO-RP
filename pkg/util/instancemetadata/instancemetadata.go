package instancemetadata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type InstanceMetadata interface {
	SubscriptionID() string
	Location() string
	ResourceGroup() string
}

type instanceMetadata struct {
	subscriptionID string
	location       string
	resourceGroup  string
}

func (im *instanceMetadata) SubscriptionID() string {
	return im.subscriptionID
}

func (im *instanceMetadata) Location() string {
	return im.location
}

func (im *instanceMetadata) ResourceGroup() string {
	return im.resourceGroup
}
