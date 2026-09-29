package arm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	sdkpolicyinsights "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/policyinsights/armpolicyinsights"

	mock_armpolicyinsights "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armpolicyinsights"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const (
	policyRG              = "fakeResourceGroup"
	storageAccountType    = "Microsoft.Storage/storageAccounts"
	storageAccount1       = "cluster1sa"
	storageAccount2       = "registrysa"
	testPolicyDefID       = "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyDefinitions/e2e-storage-public-access-validate"
	testPolicyAssignID    = "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyAssignments/e2e-storage-public-access-validate-assign"
	testStorageAPIVersion = "2021-09-01"
)

func TestIsMutatingEffect(t *testing.T) {
	for _, tt := range []struct {
		effect string
		want   bool
	}{
		{"append", true},
		{"Append", true},
		{"MODIFY", true},
		{"mutate", true},
		{"deny", false},
		{"denyAction", false},
		{"audit", false},
		{"auditIfNotExists", false},
		{"deployIfNotExists", false},
		{"disabled", false},
		{"manual", false},
		{"", false},
	} {
		t.Run(tt.effect, func(t *testing.T) {
			if got := isMutatingEffect(tt.effect); got != tt.want {
				t.Errorf("isMutatingEffect(%q) = %v, want %v", tt.effect, got, tt.want)
			}
		})
	}
}

func TestParsePolicyRef(t *testing.T) {
	_, log := testlog.LogForTesting(t)

	for _, tt := range []struct {
		name string
		id   string
		want *PolicyRef
	}{
		{
			name: "empty id returns nil",
			id:   "",
			want: nil,
		},
		{
			name: "subscription-scoped policy definition",
			id:   "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyDefinitions/def1",
			want: &PolicyRef{
				Name:           "def1",
				SubscriptionID: "00000000-0000-0000-0000-000000000000",
			},
		},
		{
			name: "resource-group-scoped policy assignment",
			id:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg1/providers/Microsoft.Authorization/policyAssignments/a1",
			want: &PolicyRef{
				Name:              "a1",
				SubscriptionID:    "00000000-0000-0000-0000-000000000000",
				ResourceGroupName: "rg1",
			},
		},
		{
			name: "unparseable id falls back to name-only",
			id:   "not-a-resource-id",
			want: &PolicyRef{Name: "not-a-resource-id"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePolicyRef(log, tt.id)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePolicyRef(%q) = %+v, want %+v", tt.id, got, tt.want)
			}
		})
	}
}

func makeFieldRestrictions(field, effect string, result sdkpolicyinsights.FieldRestrictionResult, values []string) *sdkpolicyinsights.FieldRestrictions {
	vs := make([]*string, 0, len(values))
	for _, v := range values {
		vs = append(vs, pointerutils.ToPtr(v))
	}
	return &sdkpolicyinsights.FieldRestrictions{
		Field: pointerutils.ToPtr(field),
		Restrictions: []*sdkpolicyinsights.FieldRestriction{
			{
				Result:       pointerutils.ToPtr(result),
				PolicyEffect: pointerutils.ToPtr(effect),
				Values:       vs,
				Policy: &sdkpolicyinsights.PolicyReference{
					PolicyDefinitionID: pointerutils.ToPtr(testPolicyDefID),
					PolicyAssignmentID: pointerutils.ToPtr(testPolicyAssignID),
				},
			},
		},
	}
}

func TestEnrichMismatchesWithPolicyContext(t *testing.T) {
	ctx := context.Background()

	baseMismatch := func(name string) Mismatch {
		return Mismatch{
			ResourceName:  name,
			ResourceType:  storageAccountType,
			ResourceGroup: policyRG,
			Property:      "/properties/publicNetworkAccess",
			Expected:      "Enabled",
			Actual:        "Disabled",
		}
	}

	for _, tt := range []struct {
		name       string
		mismatches []Mismatch
		mocks      func(*mock_armpolicyinsights.MockPolicyRestrictionsClient)
		wantNil    bool
		verify     func(t *testing.T, out []Mismatch)
	}{
		{
			name:       "nil client is a no-op",
			mismatches: []Mismatch{baseMismatch(storageAccount1)},
			wantNil:    true,
			verify: func(t *testing.T, out []Mismatch) {
				if out[0].Policies != nil {
					t.Errorf("expected no policies, got %#v", out[0].Policies)
				}
			},
		},
		{
			name:       "empty mismatches is a no-op",
			mismatches: nil,
			mocks:      func(c *mock_armpolicyinsights.MockPolicyRestrictionsClient) {},
			verify:     func(t *testing.T, out []Mismatch) {},
		},
		{
			name:       "missing resourceName or resourceType is skipped",
			mismatches: []Mismatch{{Property: "/properties/publicNetworkAccess", Expected: "Enabled", Actual: "Disabled"}},
			mocks:      func(c *mock_armpolicyinsights.MockPolicyRestrictionsClient) {},
			verify: func(t *testing.T, out []Mismatch) {
				if out[0].Policies != nil {
					t.Errorf("expected no policies, got %#v", out[0].Policies)
				}
			},
		},
		{
			name:       "unknown resource type has no api version, entry skipped",
			mismatches: []Mismatch{{ResourceName: "foo", ResourceType: "Microsoft.Made/upType", Property: "/x", Expected: "A", Actual: "B"}},
			mocks:      func(c *mock_armpolicyinsights.MockPolicyRestrictionsClient) {},
			verify: func(t *testing.T, out []Mismatch) {
				if out[0].Policies != nil {
					t.Errorf("expected no policies for unknown type, got %#v", out[0].Policies)
				}
			},
		},
		{
			name:       "mutating effect is attached",
			mismatches: []Mismatch{baseMismatch(storageAccount1)},
			mocks: func(c *mock_armpolicyinsights.MockPolicyRestrictionsClient) {
				c.EXPECT().
					CheckAtResourceGroupScope(gomock.Any(), policyRG, gomock.Any(), gomock.Nil()).
					Return(sdkpolicyinsights.PolicyRestrictionsClientCheckAtResourceGroupScopeResponse{
						CheckRestrictionsResult: sdkpolicyinsights.CheckRestrictionsResult{
							FieldRestrictions: []*sdkpolicyinsights.FieldRestrictions{
								makeFieldRestrictions("Microsoft.Storage/storageAccounts/publicNetworkAccess", "modify", sdkpolicyinsights.FieldRestrictionResultRemoved, []string{"Disabled"}),
							},
						},
					}, nil)
			},
			verify: func(t *testing.T, out []Mismatch) {
				if len(out[0].Policies) != 1 {
					t.Fatalf("expected 1 policy, got %d: %#v", len(out[0].Policies), out[0].Policies)
				}
				p := out[0].Policies[0]
				if p.Effect != "modify" {
					t.Errorf("Effect = %q, want modify", p.Effect)
				}
				if p.Field != "Microsoft.Storage/storageAccounts/publicNetworkAccess" {
					t.Errorf("Field = %q", p.Field)
				}
				if p.Result != string(sdkpolicyinsights.FieldRestrictionResultRemoved) {
					t.Errorf("Result = %q", p.Result)
				}
				if len(p.Values) != 1 || p.Values[0] != "Disabled" {
					t.Errorf("Values = %#v, want [Disabled]", p.Values)
				}
				if p.PolicyDefinition == nil || p.PolicyDefinition.Name != "e2e-storage-public-access-validate" {
					t.Errorf("PolicyDefinition = %#v", p.PolicyDefinition)
				}
				if p.PolicyAssignment == nil || p.PolicyAssignment.Name != "e2e-storage-public-access-validate-assign" {
					t.Errorf("PolicyAssignment = %#v", p.PolicyAssignment)
				}
			},
		},
		{
			name:       "non-mutating effect is filtered out",
			mismatches: []Mismatch{baseMismatch(storageAccount1)},
			mocks: func(c *mock_armpolicyinsights.MockPolicyRestrictionsClient) {
				c.EXPECT().
					CheckAtResourceGroupScope(gomock.Any(), policyRG, gomock.Any(), gomock.Nil()).
					Return(sdkpolicyinsights.PolicyRestrictionsClientCheckAtResourceGroupScopeResponse{
						CheckRestrictionsResult: sdkpolicyinsights.CheckRestrictionsResult{
							FieldRestrictions: []*sdkpolicyinsights.FieldRestrictions{
								makeFieldRestrictions("Microsoft.Storage/storageAccounts/publicNetworkAccess", "deny", sdkpolicyinsights.FieldRestrictionResultDeny, nil),
							},
						},
					}, nil)
			},
			verify: func(t *testing.T, out []Mismatch) {
				if out[0].Policies != nil {
					t.Errorf("expected policies filtered to empty, got %#v", out[0].Policies)
				}
			},
		},
		{
			name:       "api error leaves entry untouched",
			mismatches: []Mismatch{baseMismatch(storageAccount1)},
			mocks: func(c *mock_armpolicyinsights.MockPolicyRestrictionsClient) {
				c.EXPECT().
					CheckAtResourceGroupScope(gomock.Any(), policyRG, gomock.Any(), gomock.Nil()).
					Return(sdkpolicyinsights.PolicyRestrictionsClientCheckAtResourceGroupScopeResponse{}, errors.New("boom"))
			},
			verify: func(t *testing.T, out []Mismatch) {
				if out[0].Policies != nil {
					t.Errorf("expected no policies on error, got %#v", out[0].Policies)
				}
			},
		},
		{
			name:       "same resource cached across multiple mismatches",
			mismatches: []Mismatch{baseMismatch(storageAccount1), baseMismatch(storageAccount1)},
			mocks: func(c *mock_armpolicyinsights.MockPolicyRestrictionsClient) {
				c.EXPECT().
					CheckAtResourceGroupScope(gomock.Any(), policyRG, gomock.Any(), gomock.Nil()).
					Times(1).
					Return(sdkpolicyinsights.PolicyRestrictionsClientCheckAtResourceGroupScopeResponse{
						CheckRestrictionsResult: sdkpolicyinsights.CheckRestrictionsResult{
							FieldRestrictions: []*sdkpolicyinsights.FieldRestrictions{
								makeFieldRestrictions("Microsoft.Storage/storageAccounts/publicNetworkAccess", "append", sdkpolicyinsights.FieldRestrictionResultRequired, []string{"Enabled"}),
							},
						},
					}, nil)
			},
			verify: func(t *testing.T, out []Mismatch) {
				for i, m := range out {
					if len(m.Policies) != 1 {
						t.Fatalf("mismatch %d: expected 1 policy, got %d", i, len(m.Policies))
					}
				}
			},
		},
		{
			name:       "distinct resources each get their own call",
			mismatches: []Mismatch{baseMismatch(storageAccount1), baseMismatch(storageAccount2)},
			mocks: func(c *mock_armpolicyinsights.MockPolicyRestrictionsClient) {
				c.EXPECT().
					CheckAtResourceGroupScope(gomock.Any(), policyRG, gomock.Any(), gomock.Nil()).
					Times(2).
					Return(sdkpolicyinsights.PolicyRestrictionsClientCheckAtResourceGroupScopeResponse{
						CheckRestrictionsResult: sdkpolicyinsights.CheckRestrictionsResult{
							FieldRestrictions: []*sdkpolicyinsights.FieldRestrictions{
								makeFieldRestrictions("Microsoft.Storage/storageAccounts/publicNetworkAccess", "modify", sdkpolicyinsights.FieldRestrictionResultRemoved, nil),
							},
						},
					}, nil)
			},
			verify: func(t *testing.T, out []Mismatch) {
				for i, m := range out {
					if len(m.Policies) != 1 {
						t.Errorf("mismatch %d: expected 1 policy, got %d", i, len(m.Policies))
					}
				}
			},
		},
		{
			name:       "restriction on unrelated field is not attached",
			mismatches: []Mismatch{baseMismatch(storageAccount1)},
			mocks: func(c *mock_armpolicyinsights.MockPolicyRestrictionsClient) {
				c.EXPECT().
					CheckAtResourceGroupScope(gomock.Any(), policyRG, gomock.Any(), gomock.Nil()).
					Return(sdkpolicyinsights.PolicyRestrictionsClientCheckAtResourceGroupScopeResponse{
						CheckRestrictionsResult: sdkpolicyinsights.CheckRestrictionsResult{
							FieldRestrictions: []*sdkpolicyinsights.FieldRestrictions{
								makeFieldRestrictions("Microsoft.Storage/storageAccounts/allowBlobPublicAccess", "modify", sdkpolicyinsights.FieldRestrictionResultRemoved, nil),
								makeFieldRestrictions("Microsoft.Storage/storageAccounts/publicNetworkAccess", "modify", sdkpolicyinsights.FieldRestrictionResultRemoved, nil),
							},
						},
					}, nil)
			},
			verify: func(t *testing.T, out []Mismatch) {
				if len(out[0].Policies) != 1 {
					t.Fatalf("expected 1 policy, got %d: %#v", len(out[0].Policies), out[0].Policies)
				}
				if got := out[0].Policies[0].Field; got != "Microsoft.Storage/storageAccounts/publicNetworkAccess" {
					t.Errorf("attached wrong field: %q", got)
				}
			},
		},
		{
			name: "different properties on same resource pick their own attribution with one API call",
			mismatches: []Mismatch{
				baseMismatch(storageAccount1),
				func() Mismatch {
					m := baseMismatch(storageAccount1)
					m.Property = "/properties/allowBlobPublicAccess"
					return m
				}(),
			},
			mocks: func(c *mock_armpolicyinsights.MockPolicyRestrictionsClient) {
				c.EXPECT().
					CheckAtResourceGroupScope(gomock.Any(), policyRG, gomock.Any(), gomock.Nil()).
					Times(1).
					Return(sdkpolicyinsights.PolicyRestrictionsClientCheckAtResourceGroupScopeResponse{
						CheckRestrictionsResult: sdkpolicyinsights.CheckRestrictionsResult{
							FieldRestrictions: []*sdkpolicyinsights.FieldRestrictions{
								makeFieldRestrictions("Microsoft.Storage/storageAccounts/publicNetworkAccess", "modify", sdkpolicyinsights.FieldRestrictionResultRemoved, nil),
								makeFieldRestrictions("properties.allowBlobPublicAccess", "append", sdkpolicyinsights.FieldRestrictionResultRequired, []string{"false"}),
							},
						},
					}, nil)
			},
			verify: func(t *testing.T, out []Mismatch) {
				if len(out[0].Policies) != 1 || out[0].Policies[0].Field != "Microsoft.Storage/storageAccounts/publicNetworkAccess" {
					t.Errorf("mismatch 0 attribution wrong: %#v", out[0].Policies)
				}
				if len(out[1].Policies) != 1 || out[1].Policies[0].Field != "properties.allowBlobPublicAccess" {
					t.Errorf("mismatch 1 attribution wrong: %#v", out[1].Policies)
				}
			},
		},
		{
			name: "case-insensitive field vs property match",
			mismatches: []Mismatch{{
				ResourceName:  storageAccount1,
				ResourceType:  storageAccountType,
				ResourceGroup: policyRG,
				Property:      "/Properties/PublicNetworkAccess",
				Expected:      "Enabled",
				Actual:        "Disabled",
			}},
			mocks: func(c *mock_armpolicyinsights.MockPolicyRestrictionsClient) {
				c.EXPECT().
					CheckAtResourceGroupScope(gomock.Any(), policyRG, gomock.Any(), gomock.Nil()).
					Return(sdkpolicyinsights.PolicyRestrictionsClientCheckAtResourceGroupScopeResponse{
						CheckRestrictionsResult: sdkpolicyinsights.CheckRestrictionsResult{
							FieldRestrictions: []*sdkpolicyinsights.FieldRestrictions{
								makeFieldRestrictions("microsoft.storage/storageaccounts/publicnetworkaccess", "modify", sdkpolicyinsights.FieldRestrictionResultRemoved, nil),
							},
						},
					}, nil)
			},
			verify: func(t *testing.T, out []Mismatch) {
				if len(out[0].Policies) != 1 {
					t.Errorf("expected 1 policy despite case difference, got %d", len(out[0].Policies))
				}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			_, log := testlog.LogForTesting(t)

			var client *mock_armpolicyinsights.MockPolicyRestrictionsClient
			if !tt.wantNil {
				client = mock_armpolicyinsights.NewMockPolicyRestrictionsClient(controller)
				if tt.mocks != nil {
					tt.mocks(client)
				}
			}

			mismatches := append([]Mismatch(nil), tt.mismatches...)
			if tt.wantNil {
				EnrichMismatchesWithPolicyContext(ctx, log, nil, policyRG, mismatches)
			} else {
				EnrichMismatchesWithPolicyContext(ctx, log, client, policyRG, mismatches)
			}

			if tt.verify != nil {
				tt.verify(t, mismatches)
			}
		})
	}
}

func TestEnrichMismatchesRequestShape(t *testing.T) {
	ctx := context.Background()
	controller := gomock.NewController(t)
	_, log := testlog.LogForTesting(t)
	client := mock_armpolicyinsights.NewMockPolicyRestrictionsClient(controller)

	var gotReq sdkpolicyinsights.CheckRestrictionsRequest
	client.EXPECT().
		CheckAtResourceGroupScope(gomock.Any(), policyRG, gomock.Any(), gomock.Nil()).
		DoAndReturn(func(_ context.Context, _ string, req sdkpolicyinsights.CheckRestrictionsRequest, _ *sdkpolicyinsights.PolicyRestrictionsClientCheckAtResourceGroupScopeOptions) (sdkpolicyinsights.PolicyRestrictionsClientCheckAtResourceGroupScopeResponse, error) {
			gotReq = req
			return sdkpolicyinsights.PolicyRestrictionsClientCheckAtResourceGroupScopeResponse{}, nil
		})

	mismatches := []Mismatch{{
		ResourceName: storageAccount1,
		ResourceType: storageAccountType,
	}}
	EnrichMismatchesWithPolicyContext(ctx, log, client, policyRG, mismatches)

	if gotReq.ResourceDetails == nil {
		t.Fatalf("ResourceDetails is nil")
	}
	if gotReq.ResourceDetails.APIVersion == nil || *gotReq.ResourceDetails.APIVersion != testStorageAPIVersion {
		t.Errorf("APIVersion = %v, want %s", gotReq.ResourceDetails.APIVersion, testStorageAPIVersion)
	}
	content, ok := gotReq.ResourceDetails.ResourceContent.(map[string]interface{})
	if !ok {
		t.Fatalf("ResourceContent is not map[string]interface{}: %T", gotReq.ResourceDetails.ResourceContent)
	}
	if content["type"] != storageAccountType {
		t.Errorf("content.type = %v, want %s", content["type"], storageAccountType)
	}
	if content["name"] != storageAccount1 {
		t.Errorf("content.name = %v, want %s", content["name"], storageAccount1)
	}
}
