package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestAuditTargetResourceData(t *testing.T) {
	var (
		location                  = "test-location"
		resourceGroupName         = "test-resource-group-name"
		resourceProviderNamespace = "test-resource-provider-namespace"
		resourceName              = "test-resource-name"
		resourceType              = "test-resource-type"
		subscriptionID            = "test-subscription"
		operationID               = "test-operation-id"
	)

	var testCases = []struct {
		url          string
		expectedKind string
		expectedName string
	}{
		{
			url:          fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
			expectedKind: resourceType,
			expectedName: fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
		},
		{
			url:          fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/%s/%s", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType),
			expectedKind: resourceType,
			expectedName: fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/%s/%s", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType),
		},
		{
			url:          fmt.Sprintf("/subscriptions/%s/providers/%s/%s", subscriptionID, resourceProviderNamespace, resourceType),
			expectedKind: resourceType,
			expectedName: fmt.Sprintf("/subscriptions/%s/providers/%s/%s", subscriptionID, resourceProviderNamespace, resourceType),
		},
		{
			url:          fmt.Sprintf("/subscriptions/%s/providers/%s/locations/%s/operationsstatus/%s", subscriptionID, resourceProviderNamespace, location, operationID),
			expectedKind: "",
			expectedName: fmt.Sprintf("/subscriptions/%s/providers/%s/locations/%s/operationsstatus/%s", subscriptionID, resourceProviderNamespace, location, operationID),
		},
		{
			url:          fmt.Sprintf("/subscriptions/%s/providers/%s/locations/%s/operationresults/%s", subscriptionID, resourceProviderNamespace, location, operationID),
			expectedKind: "",
			expectedName: fmt.Sprintf("/subscriptions/%s/providers/%s/locations/%s/operationresults/%s", subscriptionID, resourceProviderNamespace, location, operationID),
		},
		{
			url:          fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s/listcredentials", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
			expectedKind: resourceType,
			expectedName: fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s/listcredentials", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
		},
		{
			url:          fmt.Sprintf("/admin/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s/kubernetesobjects", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
			expectedKind: resourceType,
			expectedName: fmt.Sprintf("/admin/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s/kubernetesobjects", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
		},
		{
			url:          fmt.Sprintf("/admin/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s/resources", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
			expectedKind: resourceType,
			expectedName: fmt.Sprintf("/admin/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s/resources", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
		},
		{
			url:          fmt.Sprintf("/admin/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s/serialconsole", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
			expectedKind: resourceType,
			expectedName: fmt.Sprintf("/admin/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s/serialconsole", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
		},
		{
			url:          fmt.Sprintf("/admin/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s/redeployvm", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
			expectedKind: resourceType,
			expectedName: fmt.Sprintf("/admin/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s/redeployvm", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
		},
		{
			url:          fmt.Sprintf("/admin/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s/upgrade", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
			expectedKind: resourceType,
			expectedName: fmt.Sprintf("/admin/subscriptions/%s/resourcegroups/%s/providers/%s/%s/%s/upgrade", subscriptionID, resourceGroupName, resourceProviderNamespace, resourceType, resourceName),
		},
		{
			url:          fmt.Sprintf("/admin/providers/%s/%s", resourceProviderNamespace, resourceType),
			expectedKind: resourceType,
			expectedName: fmt.Sprintf("/admin/providers/%s/%s", resourceProviderNamespace, resourceType),
		},
		{
			url:          fmt.Sprintf("/providers/%s/operations", resourceProviderNamespace),
			expectedKind: "",
			expectedName: fmt.Sprintf("/providers/%s/operations", resourceProviderNamespace),
		},
		{
			url:          fmt.Sprintf("/subscriptions/%s", subscriptionID),
			expectedKind: "",
			expectedName: fmt.Sprintf("/subscriptions/%s", subscriptionID),
		},
	}

	for _, tc := range testCases {
		parsedURL, err := url.Parse(tc.url)
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		request := &http.Request{URL: parsedURL}
		actualKind := auditTargetResourceType(request)
		actualName := request.URL.Path
		if tc.expectedKind != actualKind {
			t.Errorf("%s: expected %s, actual: %s", tc.url, tc.expectedKind, actualKind)
		}

		if tc.expectedName != actualName {
			t.Errorf("%s: expected %s, actual: %s", tc.url, tc.expectedName, actualName)
		}
	}
}

func TestIsAdminOp(t *testing.T) {
	var testCases = []struct {
		url      string
		expected bool
	}{
		{url: "", expected: false},
		{url: "/", expected: false},
		{url: "/foo", expected: false},
		{url: "/foo/bar", expected: false},
		{url: "/admin", expected: true},
		{url: "/admin/foo", expected: true},
	}

	for _, tc := range testCases {
		parsedURL, err := url.Parse(tc.url)
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		r := &http.Request{URL: parsedURL}
		if actual := isAdminOp(r); tc.expected != actual {
			t.Errorf("%s: expected: %t, actual: %t", tc.url, tc.expected, actual)
		}
	}
}
