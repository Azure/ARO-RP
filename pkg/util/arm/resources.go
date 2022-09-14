package arm

import (
	"fmt"
	"regexp"
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
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s", r.SubscriptionID, r.ResourceGroup, r.Provider, r.ResourceType, r.ResourceName)
}

// String function returns a string in form of azureResourceID
func (r ArmResource) String() string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s/%s/%s", r.SubscriptionID, r.ResourceGroup, r.Provider, r.ResourceType, r.ResourceName, r.SubResource.ResourceType, r.SubResource.ResourceName)
}

// ParseArmResourceId take the resourceID of a child resource to an OpenShiftCluster
// TODO refactor this function to support an additional layer of child resources if we ever get to that point, right now only supports 1 child resource
func ParseArmResourceId(resourceId string) (*ArmResource, error) {
	const resourceIDPatternText = `(?i)subscriptions/(.+)/resourceGroups/(.+)/providers/(.+?)/(.+?)/(.+?)/(.+?)/(.+)`
	resourceIDPattern := regexp.MustCompile(resourceIDPatternText)
	match := resourceIDPattern.FindStringSubmatch(resourceId)

	if len(match) != 8 || strings.Contains(match[7], "/") {
		return nil, fmt.Errorf("parsing failed for %s. Invalid resource Id format", resourceId)
	}

	result := &ArmResource{
		SubscriptionID: match[1],
		ResourceGroup:  match[2],
		Provider:       match[3],
		ResourceType:   match[4],
		ResourceName:   match[5],
		SubResource: SubResource{
			ResourceType: match[6],
			ResourceName: match[7],
		},
	}
	return result, nil
}
