package arm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	sdkarm "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const (
	whatifResourceGroup = "fakeResourceGroup"
	whatifDeploymentSA1 = "cluster1sa"
	whatifDeploymentSA2 = "registrysa"
)

func storageAccountResourceID(name string) string {
	return "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + whatifResourceGroup +
		"/providers/Microsoft.Storage/storageAccounts/" + name
}

func TestBuildStorageAccountsValidationMap(t *testing.T) {
	expectedProps := map[string]interface{}{
		"/properties/publicNetworkAccess": "Enabled",
	}

	for _, tt := range []struct {
		name  string
		input []string
		want  map[string]map[string]interface{}
	}{
		{
			name:  "empty input returns empty map",
			input: nil,
			want:  map[string]map[string]interface{}{},
		},
		{
			name:  "skips empty account names",
			input: []string{"", ""},
			want:  map[string]map[string]interface{}{},
		},
		{
			name:  "single account name is lowercased",
			input: []string{"MyStorageAccount"},
			want: map[string]map[string]interface{}{
				"mystorageaccount": {
					"resourceType":       "Microsoft.Storage/storageAccounts",
					"expectedProperties": expectedProps,
				},
			},
		},
		{
			name:  "multiple account names, some empty, all lowercased",
			input: []string{"ClusterSA", "", "RegistrySA"},
			want: map[string]map[string]interface{}{
				"clustersa": {
					"resourceType":       "Microsoft.Storage/storageAccounts",
					"expectedProperties": expectedProps,
				},
				"registrysa": {
					"resourceType":       "Microsoft.Storage/storageAccounts",
					"expectedProperties": expectedProps,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildStorageAccountsValidationMap(tt.input...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   string
		want string
	}{
		{"already normalized", "/properties/publicNetworkAccess", "/properties/publicNetworkAccess"},
		{"dot notation", "properties.publicNetworkAccess", "/properties/publicNetworkAccess"},
		{"single segment dot notation", "publicNetworkAccess", "/publicNetworkAccess"},
		{"empty", "", "/"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizePath(tt.in); got != tt.want {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFindMatchingResourceConfig(t *testing.T) {
	expectedProps := map[string]interface{}{
		"/properties/publicNetworkAccess": "Enabled",
	}
	validationMap := BuildStorageAccountsValidationMap(whatifDeploymentSA1)

	for _, tt := range []struct {
		name       string
		resourceID string
		want       map[string]interface{}
	}{
		{
			name:       "matching storage account (case-insensitive) returns expected properties",
			resourceID: storageAccountResourceID(strings.ToUpper(whatifDeploymentSA1)),
			want:       expectedProps,
		},
		{
			name:       "unknown storage account returns nil",
			resourceID: storageAccountResourceID("otheraccount"),
			want:       nil,
		},
		{
			name:       "matching name but wrong resource type returns nil",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/" + whatifDeploymentSA1,
			want:       nil,
		},
		{
			name:       "sub-resource type is not a prefix match (exact match required)",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/" + whatifDeploymentSA1 + "/blobServices/default",
			want:       nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := sdkarm.ParseResourceID(tt.resourceID)
			if err != nil {
				t.Fatalf("ParseResourceID: %v", err)
			}
			got := findMatchingResourceConfig(parsed, validationMap)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCollectPropertyChanges(t *testing.T) {
	_, log := testlog.LogForTesting(t)

	expectedProps := map[string]interface{}{
		"/properties/publicNetworkAccess": "Enabled",
	}

	parsed, err := sdkarm.ParseResourceID(storageAccountResourceID(whatifDeploymentSA1))
	if err != nil {
		t.Fatalf("ParseResourceID: %v", err)
	}

	for _, tt := range []struct {
		name          string
		changes       []mgmtfeatures.WhatIfPropertyChange
		wantMismatch  bool
		wantMismatchN int
	}{
		{
			name: "matching value produces no mismatch",
			changes: []mgmtfeatures.WhatIfPropertyChange{{
				Path:   pointerutils.ToPtr("/properties/publicNetworkAccess"),
				Before: "Enabled",
				After:  "Disabled",
			}},
		},
		{
			name: "mismatched value is reported",
			changes: []mgmtfeatures.WhatIfPropertyChange{{
				Path:   pointerutils.ToPtr("/properties/publicNetworkAccess"),
				Before: "Disabled",
				After:  "Enabled",
			}},
			wantMismatch:  true,
			wantMismatchN: 1,
		},
		{
			name: "nil Before is reported as mismatch",
			changes: []mgmtfeatures.WhatIfPropertyChange{{
				Path:   pointerutils.ToPtr("/properties/publicNetworkAccess"),
				Before: nil,
				After:  "Enabled",
			}},
			wantMismatch:  true,
			wantMismatchN: 1,
		},
		{
			name: "unwatched path is ignored",
			changes: []mgmtfeatures.WhatIfPropertyChange{{
				Path:   pointerutils.ToPtr("/properties/somethingElse"),
				Before: "value1",
				After:  "value2",
			}},
		},
		{
			name: "dot-notation path is normalized then matched",
			changes: []mgmtfeatures.WhatIfPropertyChange{{
				Path:   pointerutils.ToPtr("properties.publicNetworkAccess"),
				Before: "Disabled",
				After:  "Enabled",
			}},
			wantMismatch:  true,
			wantMismatchN: 1,
		},
		{
			name: "nil path is skipped",
			changes: []mgmtfeatures.WhatIfPropertyChange{{
				Path:   nil,
				Before: "Disabled",
			}},
		},
		{
			name: "children are recursed",
			changes: []mgmtfeatures.WhatIfPropertyChange{{
				Path: pointerutils.ToPtr("/properties"),
				Children: &[]mgmtfeatures.WhatIfPropertyChange{{
					Path:   pointerutils.ToPtr("/properties/publicNetworkAccess"),
					Before: "Disabled",
					After:  "Enabled",
				}},
			}},
			wantMismatch:  true,
			wantMismatchN: 1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := collectPropertyChanges(log, tt.changes, parsed, expectedProps)
			if tt.wantMismatch {
				if len(got) != tt.wantMismatchN {
					t.Fatalf("got %d mismatches, want %d: %#v", len(got), tt.wantMismatchN, got)
				}
				m := got[0]
				if m.ResourceName != whatifDeploymentSA1 {
					t.Errorf("ResourceName = %v, want %v", m.ResourceName, whatifDeploymentSA1)
				}
				if m.ResourceGroup != whatifResourceGroup {
					t.Errorf("ResourceGroup = %v, want %v", m.ResourceGroup, whatifResourceGroup)
				}
				if m.Expected != "Enabled" {
					t.Errorf("Expected = %v, want Enabled", m.Expected)
				}
				if m.Property != "/properties/publicNetworkAccess" {
					t.Errorf("Property = %v, want /properties/publicNetworkAccess", m.Property)
				}
			} else if len(got) != 0 {
				t.Errorf("expected no mismatches, got %#v", got)
			}
		})
	}
}

func TestValidateDeploymentWithWhatIf(t *testing.T) {
	ctx := context.Background()

	template := &Template{}
	validationMap := BuildStorageAccountsValidationMap(whatifDeploymentSA1, whatifDeploymentSA2)

	whatIfParams := mgmtfeatures.DeploymentWhatIf{
		Properties: &mgmtfeatures.DeploymentWhatIfProperties{
			Template: template,
			Mode:     mgmtfeatures.Incremental,
		},
	}

	mismatchChange := mgmtfeatures.WhatIfChange{
		ResourceID: pointerutils.ToPtr(storageAccountResourceID(whatifDeploymentSA1)),
		Delta: &[]mgmtfeatures.WhatIfPropertyChange{{
			Path:   pointerutils.ToPtr("/properties/publicNetworkAccess"),
			Before: "Disabled",
			After:  "Enabled",
		}},
	}

	matchingChange := mgmtfeatures.WhatIfChange{
		ResourceID: pointerutils.ToPtr(storageAccountResourceID(whatifDeploymentSA1)),
		Delta: &[]mgmtfeatures.WhatIfPropertyChange{{
			Path:   pointerutils.ToPtr("/properties/publicNetworkAccess"),
			Before: "Enabled",
			After:  "Enabled",
		}},
	}

	unknownResourceChange := mgmtfeatures.WhatIfChange{
		ResourceID: pointerutils.ToPtr(storageAccountResourceID("unmonitoredsa")),
		Delta: &[]mgmtfeatures.WhatIfPropertyChange{{
			Path:   pointerutils.ToPtr("/properties/publicNetworkAccess"),
			Before: "Disabled",
		}},
	}

	unparseableChange := mgmtfeatures.WhatIfChange{
		ResourceID: pointerutils.ToPtr("not-a-resource-id"),
		Delta: &[]mgmtfeatures.WhatIfPropertyChange{{
			Path:   pointerutils.ToPtr("/properties/publicNetworkAccess"),
			Before: "Disabled",
		}},
	}

	nilDeltaChange := mgmtfeatures.WhatIfChange{
		ResourceID: pointerutils.ToPtr(storageAccountResourceID(whatifDeploymentSA1)),
		Delta:      nil,
	}

	for _, tt := range []struct {
		name    string
		mocks   func(*mock_features.MockDeploymentsClient)
		wantErr string
	}{
		{
			name: "WhatIf call error is swallowed and nil returned",
			mocks: func(dc *mock_features.MockDeploymentsClient) {
				dc.EXPECT().
					WhatIfAndWait(ctx, whatifResourceGroup, deploymentName, whatIfParams).
					Return(mgmtfeatures.WhatIfOperationResult{}, errors.New("whatif failed"))
			},
		},
		{
			name: "nil WhatIfOperationProperties returns nil",
			mocks: func(dc *mock_features.MockDeploymentsClient) {
				dc.EXPECT().
					WhatIfAndWait(ctx, whatifResourceGroup, deploymentName, whatIfParams).
					Return(mgmtfeatures.WhatIfOperationResult{}, nil)
			},
		},
		{
			name: "nil Changes returns nil",
			mocks: func(dc *mock_features.MockDeploymentsClient) {
				dc.EXPECT().
					WhatIfAndWait(ctx, whatifResourceGroup, deploymentName, whatIfParams).
					Return(mgmtfeatures.WhatIfOperationResult{
						WhatIfOperationProperties: &mgmtfeatures.WhatIfOperationProperties{},
					}, nil)
			},
		},
		{
			name: "matching value returns nil (no mismatch)",
			mocks: func(dc *mock_features.MockDeploymentsClient) {
				dc.EXPECT().
					WhatIfAndWait(ctx, whatifResourceGroup, deploymentName, whatIfParams).
					Return(mgmtfeatures.WhatIfOperationResult{
						WhatIfOperationProperties: &mgmtfeatures.WhatIfOperationProperties{
							Changes: &[]mgmtfeatures.WhatIfChange{matchingChange},
						},
					}, nil)
			},
		},
		{
			name: "unknown resource is ignored",
			mocks: func(dc *mock_features.MockDeploymentsClient) {
				dc.EXPECT().
					WhatIfAndWait(ctx, whatifResourceGroup, deploymentName, whatIfParams).
					Return(mgmtfeatures.WhatIfOperationResult{
						WhatIfOperationProperties: &mgmtfeatures.WhatIfOperationProperties{
							Changes: &[]mgmtfeatures.WhatIfChange{unknownResourceChange},
						},
					}, nil)
			},
		},
		{
			name: "unparseable resource ID is skipped",
			mocks: func(dc *mock_features.MockDeploymentsClient) {
				dc.EXPECT().
					WhatIfAndWait(ctx, whatifResourceGroup, deploymentName, whatIfParams).
					Return(mgmtfeatures.WhatIfOperationResult{
						WhatIfOperationProperties: &mgmtfeatures.WhatIfOperationProperties{
							Changes: &[]mgmtfeatures.WhatIfChange{unparseableChange},
						},
					}, nil)
			},
		},
		{
			name: "nil Delta is skipped",
			mocks: func(dc *mock_features.MockDeploymentsClient) {
				dc.EXPECT().
					WhatIfAndWait(ctx, whatifResourceGroup, deploymentName, whatIfParams).
					Return(mgmtfeatures.WhatIfOperationResult{
						WhatIfOperationProperties: &mgmtfeatures.WhatIfOperationProperties{
							Changes: &[]mgmtfeatures.WhatIfChange{nilDeltaChange},
						},
					}, nil)
			},
		},
		{
			name: "mismatched property returns DeploymentFailed CloudError",
			mocks: func(dc *mock_features.MockDeploymentsClient) {
				dc.EXPECT().
					WhatIfAndWait(ctx, whatifResourceGroup, deploymentName, whatIfParams).
					Return(mgmtfeatures.WhatIfOperationResult{
						WhatIfOperationProperties: &mgmtfeatures.WhatIfOperationProperties{
							Changes: &[]mgmtfeatures.WhatIfChange{mismatchChange},
						},
					}, nil)
			},
			wantErr: `400: DeploymentFailed: : Deployment failed. Details: : : Unexpected property mutations detected, likely due to Azure policies. Details: [{"resourceName":"cluster1sa","resourceType":"Microsoft.Storage/storageAccounts","resourceGroup":"fakeResourceGroup","property":"/properties/publicNetworkAccess","expected":"Enabled","actual":"Disabled"}]`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)

			deploymentsClient := mock_features.NewMockDeploymentsClient(controller)
			tt.mocks(deploymentsClient)

			_, log := testlog.LogForTesting(t)

			mismatches := ValidateDeploymentWithWhatIf(ctx, log, deploymentsClient, whatifResourceGroup, deploymentName, template, validationMap)
			err := MismatchesToCloudError(mismatches)

			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.wantErr != "" {
				ce, ok := err.(*api.CloudError)
				if !ok {
					t.Fatalf("expected *api.CloudError, got %T", err)
				}
				if ce.StatusCode != http.StatusBadRequest {
					t.Errorf("StatusCode = %d, want %d", ce.StatusCode, http.StatusBadRequest)
				}
				if ce.Code != api.CloudErrorCodeDeploymentFailed {
					t.Errorf("Code = %q, want %q", ce.Code, api.CloudErrorCodeDeploymentFailed)
				}
			} else if len(mismatches) != 0 {
				t.Errorf("expected no mismatches, got %#v", mismatches)
			}
		})
	}
}
