package arm

import (
	"fmt"
	"strings"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// ArmResource represents a resource and its child resources.
// Typically we would use the autorest package for this, but
// It does not have support for subresources
type ArmResource struct {
	SubscriptionID string
	ResourceGroup  string
	Provider       string
	ResourceName   string
	ResourceType   string
	SubResource    SubResource
}

// SubResource represents an ARM Proxy Resource
// ARM supports up to 3 levels of nested resources
// https://eng.ms/docs/products/arm/api_contracts/guidelines/rpc#rpc030-avoid-excessive-resource-type-nesting
type SubResource struct {
	ResourceName string
	ResourceType string
	SubResource  *SubResource
}

// ParentResourcetoString returns a string of the parent object in form of azureResourceID
func (r ArmResource) ParentResource() string {
	return fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s", r.SubscriptionID, r.ResourceGroup, r.Provider, r.ResourceType, r.ResourceName)
}

// String function returns a string in form of azureResourceID
func (r ArmResource) String() string {
	return fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s/%s/%s", r.SubscriptionID, r.ResourceGroup, r.Provider, r.ResourceType, r.ResourceName, r.SubResource.ResourceType, r.SubResource.ResourceName)
}

// ParseArmResourceId takes in an ARM resource ID and returns an ArmResource object representing that resource. It supports up to two levels of subresource nesting.
func ParseArmResourceId(resourceId string) (*ArmResource, error) {
	resourceComponents := strings.Split(strings.TrimPrefix(resourceId, "/"), "/")
	if len(resourceComponents) < 8 || !strings.EqualFold(resourceComponents[0], "subscriptions") || !strings.EqualFold(resourceComponents[2], "resourceGroups") || !strings.EqualFold(resourceComponents[4], "providers") {
		return nil, fmt.Errorf("parsing failed for %s. Invalid resource Id format", resourceId)
	}

	result := &ArmResource{
		SubscriptionID: resourceComponents[1],
		ResourceGroup:  resourceComponents[3],
		Provider:       resourceComponents[5],
		ResourceType:   resourceComponents[6],
		ResourceName:   resourceComponents[7],
	}
	if len(resourceComponents) > 8 {
		result.SubResource = SubResource{
			ResourceType: resourceComponents[8],
			ResourceName: resourceComponents[9],
		}
		if len(resourceComponents) > 10 {
			result.SubResource.SubResource = &SubResource{
				ResourceType: resourceComponents[8], // same subresource type as the first subresource
				ResourceName: resourceComponents[10],
			}
		}
	}
	return result, nil
}
