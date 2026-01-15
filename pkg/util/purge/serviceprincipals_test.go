package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func TestDetermineResourceGroupName(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		want        string
	}{
		{
			name:        "service principal with aro- prefix",
			displayName: "aro-v4-e2e-V123456789-eastus",
			want:        "v4-e2e-V123456789-eastus",
		},
		{
			name:        "service principal with aro- prefix and miwi suffix",
			displayName: "aro-v4-e2e-V987654321-westus-miwi",
			want:        "v4-e2e-V987654321-westus-miwi",
		},
		{
			name:        "service principal with prod-csp suffix",
			displayName: "aro-v4-e2e-V111111111-centralus-prod-csp",
			want:        "v4-e2e-V111111111-centralus-prod-csp",
		},
		{
			name:        "service principal with prod-miwi suffix",
			displayName: "aro-v4-e2e-V444444444-eastus-prod-miwi",
			want:        "v4-e2e-V444444444-eastus-prod-miwi",
		},
		{
			name:        "disk encryption set managed identity",
			displayName: "v4-e2e-V222222222-eastus-disk-encryption-set",
			want:        "v4-e2e-V222222222-eastus",
		},
		{
			name:        "disk encryption set with miwi",
			displayName: "v4-e2e-V333333333-westus-miwi-disk-encryption-set",
			want:        "v4-e2e-V333333333-westus-miwi",
		},
		{
			name:        "disk encryption set with prod-csp",
			displayName: "v4-e2e-V555555555-centralus-prod-csp-disk-encryption-set",
			want:        "v4-e2e-V555555555-centralus-prod-csp",
		},
		{
			name:        "disk encryption set with prod-miwi",
			displayName: "v4-e2e-V666666666-westeurope-prod-miwi-disk-encryption-set",
			want:        "v4-e2e-V666666666-westeurope-prod-miwi",
		},
		{
			name:        "other pattern returns as-is",
			displayName: "some-other-pattern",
			want:        "some-other-pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineResourceGroupName(tt.displayName)
			if got != tt.want {
				t.Errorf("determineResourceGroupName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildIDPattern(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		shouldMatch bool
	}{
		{
			name:        "valid 9-digit build ID",
			displayName: "aro-v4-e2e-V123456789-eastus",
			shouldMatch: true,
		},
		{
			name:        "valid build ID in middle of string",
			displayName: "prefix-V987654321-suffix",
			shouldMatch: true,
		},
		{
			name:        "no build ID",
			displayName: "aro-infrastructure-sp",
			shouldMatch: false,
		},
		{
			name:        "lowercase v",
			displayName: "aro-v4-e2e-v123456789-eastus",
			shouldMatch: false,
		},
		{
			name:        "build ID with letters",
			displayName: "aro-v4-e2e-V12345678a-eastus",
			shouldMatch: false,
		},
		{
			name:        "mock-msi service principal",
			displayName: "mock-msi-aBc123",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := buildIDPattern.MatchString(tt.displayName)
			if matched != tt.shouldMatch {
				t.Errorf("buildIDPattern.MatchString(%q) = %v, want %v", tt.displayName, matched, tt.shouldMatch)
			}
		})
	}
}

func TestCheckSPNeededBasedOnRGStatus(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.New())
	ttl := 48 * time.Hour
	now := time.Now()

	tests := []struct {
		name              string
		resourceGroupName string
		mockSetup         func(*mock_features.MockResourceGroupsClient)
		wantKeep          bool
		wantReason        string
	}{
		{
			name:              "resource group not found - should delete SP",
			resourceGroupName: "v4-e2e-V123456789-eastus",
			mockSetup: func(m *mock_features.MockResourceGroupsClient) {
				notFoundErr := autorest.DetailedError{
					StatusCode: http.StatusNotFound,
				}
				m.EXPECT().
					Get(gomock.Any(), "v4-e2e-V123456789-eastus").
					Return(mgmtfeatures.ResourceGroup{}, notFoundErr)
			},
			wantKeep:   false,
			wantReason: "Resource group 'v4-e2e-V123456789-eastus' does not exist",
		},
		{
			name:              "resource group has persist tag - should keep SP",
			resourceGroupName: "v4-e2e-V123456789-eastus",
			mockSetup: func(m *mock_features.MockResourceGroupsClient) {
				m.EXPECT().
					Get(gomock.Any(), "v4-e2e-V123456789-eastus").
					Return(mgmtfeatures.ResourceGroup{
						Tags: map[string]*string{
							"persist":   pointerutils.ToPtr("true"),
							"createdAt": pointerutils.ToPtr(now.Add(-72 * time.Hour).Format(time.RFC3339Nano)),
						},
					}, nil)
			},
			wantKeep:   true,
			wantReason: "Resource group 'v4-e2e-V123456789-eastus' has 'persist' tag",
		},
		{
			name:              "resource group has persist tag (case insensitive) - should keep SP",
			resourceGroupName: "v4-e2e-V123456789-eastus",
			mockSetup: func(m *mock_features.MockResourceGroupsClient) {
				m.EXPECT().
					Get(gomock.Any(), "v4-e2e-V123456789-eastus").
					Return(mgmtfeatures.ResourceGroup{
						Tags: map[string]*string{
							"Persist":   pointerutils.ToPtr("true"),
							"createdAt": pointerutils.ToPtr(now.Add(-72 * time.Hour).Format(time.RFC3339Nano)),
						},
					}, nil)
			},
			wantKeep:   true,
			wantReason: "Resource group 'v4-e2e-V123456789-eastus' has 'persist' tag",
		},
		{
			name:              "resource group younger than TTL - should keep SP",
			resourceGroupName: "v4-e2e-V123456789-eastus",
			mockSetup: func(m *mock_features.MockResourceGroupsClient) {
				m.EXPECT().
					Get(gomock.Any(), "v4-e2e-V123456789-eastus").
					Return(mgmtfeatures.ResourceGroup{
						Tags: map[string]*string{
							"createdAt": pointerutils.ToPtr(now.Add(-24 * time.Hour).Format(time.RFC3339Nano)),
						},
					}, nil)
			},
			wantKeep:   true,
			wantReason: "Resource group 'v4-e2e-V123456789-eastus' age 24h0m0s < TTL 48h0m0s",
		},
		{
			name:              "resource group older than TTL - should delete SP",
			resourceGroupName: "v4-e2e-V123456789-eastus",
			mockSetup: func(m *mock_features.MockResourceGroupsClient) {
				m.EXPECT().
					Get(gomock.Any(), "v4-e2e-V123456789-eastus").
					Return(mgmtfeatures.ResourceGroup{
						Tags: map[string]*string{
							"createdAt": pointerutils.ToPtr(now.Add(-72 * time.Hour).Format(time.RFC3339Nano)),
						},
					}, nil)
			},
			wantKeep:   false,
			wantReason: "Resource group 'v4-e2e-V123456789-eastus' exists but age 72h0m0s >= TTL",
		},
		{
			name:              "resource group without createdAt tag - should keep SP",
			resourceGroupName: "v4-e2e-V123456789-eastus",
			mockSetup: func(m *mock_features.MockResourceGroupsClient) {
				m.EXPECT().
					Get(gomock.Any(), "v4-e2e-V123456789-eastus").
					Return(mgmtfeatures.ResourceGroup{
						Tags: map[string]*string{
							"someOtherTag": pointerutils.ToPtr("value"),
						},
					}, nil)
			},
			wantKeep:   true,
			wantReason: "Resource group 'v4-e2e-V123456789-eastus' exists but has no createdAt tag",
		},
		{
			name:              "error checking resource group (not 404) - should keep SP",
			resourceGroupName: "v4-e2e-V123456789-eastus",
			mockSetup: func(m *mock_features.MockResourceGroupsClient) {
				serviceErr := autorest.DetailedError{
					StatusCode: http.StatusInternalServerError,
					Message:    "internal server error",
				}
				m.EXPECT().
					Get(gomock.Any(), "v4-e2e-V123456789-eastus").
					Return(mgmtfeatures.ResourceGroup{}, serviceErr)
			},
			wantKeep:   true,
			wantReason: "Error checking resource group 'v4-e2e-V123456789-eastus':",
		},
		{
			name:              "azure error without status code - should keep SP",
			resourceGroupName: "v4-e2e-V123456789-eastus",
			mockSetup: func(m *mock_features.MockResourceGroupsClient) {
				azureErr := azure.ServiceError{
					Code:    "ServiceUnavailable",
					Message: "service unavailable",
				}
				m.EXPECT().
					Get(gomock.Any(), "v4-e2e-V123456789-eastus").
					Return(mgmtfeatures.ResourceGroup{}, azureErr)
			},
			wantKeep:   true,
			wantReason: "Error checking resource group 'v4-e2e-V123456789-eastus':",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockRGClient := mock_features.NewMockResourceGroupsClient(controller)
			tt.mockSetup(mockRGClient)

			rc := &ResourceCleaner{
				log:               log,
				resourcegroupscli: mockRGClient,
			}

			gotKeep, gotReason := rc.checkSPNeededBasedOnRGStatus(ctx, tt.resourceGroupName, ttl)

			if gotKeep != tt.wantKeep {
				t.Errorf("checkSPNeededBasedOnRGStatus() keep = %v, want %v", gotKeep, tt.wantKeep)
			}

			if tt.wantReason != "" {
				if len(gotReason) < len(tt.wantReason) || gotReason[:len(tt.wantReason)] != tt.wantReason {
					t.Errorf("checkSPNeededBasedOnRGStatus() reason = %q, want prefix %q", gotReason, tt.wantReason)
				}
				if !strings.Contains(gotReason, tt.resourceGroupName) {
					t.Errorf("checkSPNeededBasedOnRGStatus() reason = %q does not contain resource group name %q", gotReason, tt.resourceGroupName)
				}
			}
		})
	}
}
