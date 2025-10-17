package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/tracing"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	mock_armmsi "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armmsi"
)

func Test_isUserAssignedIdentity(t *testing.T) {
	tests := []struct {
		name     string
		identity armmsi.Identity
		want     bool
	}{
		{
			name: "correct Type",
			identity: armmsi.Identity{
				Type: to.Ptr("Microsoft.ManagedIdentity/userAssignedIdentities"),
			},
			want: true,
		},
		{
			name: "incorrect type",
			identity: armmsi.Identity{
				Type: to.Ptr("Microsoft.ManagedIdentity/SomethingWrong"),
			},
			want: false,
		},
		{
			name: "NilType",
			identity: armmsi.Identity{
				Type: nil,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUserAssignedIdentity(tt.identity)
			if got != tt.want {
				t.Errorf("isUserAssignedIdentity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isIdentityClaimed(t *testing.T) {
	timeStampLongAgo := "1512-04-12T23:20:50.52Z"
	timeStampFuture := "3023-04-12T23:20:50.52Z"

	tests := []struct {
		name     string
		identity armmsi.Identity
		want     bool
	}{
		{
			name: "timeStamp in future UTC",
			identity: armmsi.Identity{
				Tags: map[string]*string{
					ClaimedClusterTagKey:       to.Ptr("cluster"),
					ClaimedResourceGroupTagKey: to.Ptr("rg"),
					ClaimedUntilTagKey:         &timeStampFuture,
				},
			},
			want: true,
		},
		{
			name: "Invalid Timestamp",
			identity: armmsi.Identity{
				Tags: map[string]*string{
					ClaimedClusterTagKey:       to.Ptr("cluster"),
					ClaimedResourceGroupTagKey: to.Ptr("rg"),
					ClaimedUntilTagKey:         to.Ptr("invalid timestamp"),
				},
			},
			want: false,
		},
		{
			name: "nil Timestamp",
			identity: armmsi.Identity{
				Tags: map[string]*string{
					ClaimedClusterTagKey:       to.Ptr("cluster"),
					ClaimedResourceGroupTagKey: to.Ptr("rg"),
					ClaimedUntilTagKey:         nil,
				},
			},
			want: false,
		},
		{
			name: "Cluster Resource Group empty string",
			identity: armmsi.Identity{
				Tags: map[string]*string{
					ClaimedClusterTagKey:       to.Ptr("cluster"),
					ClaimedResourceGroupTagKey: to.Ptr(""),
					ClaimedUntilTagKey:         &timeStampFuture,
				},
			},
			want: false,
		},
		{
			name: "Cluster ResourceGroup Nil Pointer",
			identity: armmsi.Identity{
				Tags: map[string]*string{
					ClaimedClusterTagKey:       to.Ptr("cluster"),
					ClaimedResourceGroupTagKey: nil,
					ClaimedUntilTagKey:         &timeStampFuture,
				},
			},
			want: false,
		},
		{
			name: "Cluster Name empty string",
			identity: armmsi.Identity{
				Tags: map[string]*string{
					ClaimedClusterTagKey:       to.Ptr(""),
					ClaimedResourceGroupTagKey: to.Ptr("rg"),
					ClaimedUntilTagKey:         &timeStampFuture,
				},
			},
			want: false,
		},
		{
			name: "Cluster Name Nil Pointer",
			identity: armmsi.Identity{
				Tags: map[string]*string{
					ClaimedClusterTagKey:       nil,
					ClaimedResourceGroupTagKey: to.Ptr("rg"),
					ClaimedUntilTagKey:         &timeStampFuture,
				},
			},
			want: false,
		},
		{
			name: "timeStamp in Past UTC",
			identity: armmsi.Identity{
				Tags: map[string]*string{
					ClaimedClusterTagKey:       to.Ptr("cluster"),
					ClaimedResourceGroupTagKey: to.Ptr("rg"),
					ClaimedUntilTagKey:         &timeStampLongAgo,
				},
			},
			want: false,
		},
		{
			name: "timeStamp Future, missing Tags 1",
			identity: armmsi.Identity{
				Tags: map[string]*string{
					ClaimedClusterTagKey: to.Ptr("cluster"),
					ClaimedUntilTagKey:   &timeStampFuture,
				},
			},
			want: false,
		},
		{
			name: "timeStamp Future, missing Tags 2",
			identity: armmsi.Identity{
				Tags: map[string]*string{
					ClaimedResourceGroupTagKey: to.Ptr("rg"),
					ClaimedUntilTagKey:         &timeStampFuture,
				},
			},
			want: false,
		},
		{
			name: "missing Tags 3",
			identity: armmsi.Identity{
				Tags: map[string]*string{
					ClaimedClusterTagKey:       to.Ptr("cluster"),
					ClaimedResourceGroupTagKey: to.Ptr("rg"),
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIdentityClaimed(tt.identity)
			if got != tt.want {
				t.Errorf("isIdentityClaimed() = %v, want %v", got, tt.want)
			}
		})
	}
}

var _ = Describe("Managed Identites Pool", func() {

	defaultResourceGroup := "identityrg"
	defaultClusterResourceGroup := "clusterrg"
	defaultClusterName := "testcluster"
	defaultTimeout := time.Hour
	defaultLocation := "eastus"

	var (
		mockController *gomock.Controller
		client         *mock_armmsi.MockUserAssignedIdentitiesClient
	)

	BeforeEach(func() {
		mockController = gomock.NewController(GinkgoT())
		client = mock_armmsi.NewMockUserAssignedIdentitiesClient(mockController)
	})
	AfterEach(func() {
		mockController.Finish()
	})

	When("The identity pool is full", func() {
		It("Can't create additional identities", func() {
			// add 1000 identities
			identities := []*armmsi.Identity{}
			validUntil := time.Now().Add(defaultTimeout).Format(time.RFC3339)
			for i := 0; i < MaximumIdentitiesInPool; i++ {
				identities = append(identities, &armmsi.Identity{
					Location: &defaultLocation,
					Tags: map[string]*string{
						ClaimedResourceGroupTagKey: &defaultClusterResourceGroup,
						ClaimedClusterTagKey:       &defaultClusterName,
						ClaimedUntilTagKey:         &validUntil,
					},
					ID:   to.Ptr(IdentityId(defaultResourceGroup, fmt.Sprintf("identity-%d", i))),
					Name: to.Ptr(fmt.Sprintf("identity-%d", i)),
					Type: to.Ptr(UserAssignedIdentityType),
				})
			}
			client.EXPECT().NewListByResourceGroupPager(defaultResourceGroup, gomock.Any()).Return(identityListToPagerResult(identities))
			pool := NewManagedIdentityPool(client, defaultResourceGroup)
			createdIdentities, err := pool.ClaimIdentities(context.Background(), 10, defaultClusterResourceGroup, defaultClusterName, defaultTimeout)
			Expect(err).To(HaveOccurred())
			Expect(createdIdentities).To(BeEmpty())
		})
		It("Doesn't assign unassigned identities", func() {
			// add 999 identities
			identities := []*armmsi.Identity{}
			validUntil := time.Now().Add(defaultTimeout).Format(time.RFC3339)
			for i := 0; i < MaximumIdentitiesInPool-1; i++ {
				identities = append(identities, &armmsi.Identity{
					Location: &defaultLocation,
					Tags: map[string]*string{
						ClaimedResourceGroupTagKey: &defaultClusterResourceGroup,
						ClaimedClusterTagKey:       &defaultClusterName,
						ClaimedUntilTagKey:         &validUntil,
					},
					ID:   to.Ptr(IdentityId(defaultResourceGroup, fmt.Sprintf("identity-%d", i))),
					Name: to.Ptr(fmt.Sprintf("identity-%d", i)),
					Type: to.Ptr(UserAssignedIdentityType),
				})
			}
			identities = append(identities, &armmsi.Identity{
				Location: &defaultLocation,
				Tags: map[string]*string{
					ClaimedResourceGroupTagKey: to.Ptr(""),
					ClaimedClusterTagKey:       to.Ptr(""),
					ClaimedUntilTagKey:         to.Ptr(""),
				},
				ID:   to.Ptr(IdentityId(defaultResourceGroup, "identity-1000")),
				Name: to.Ptr("identity-1000"),
				Type: to.Ptr(UserAssignedIdentityType),
			})
			client.EXPECT().NewListByResourceGroupPager(defaultResourceGroup, gomock.Any()).Return(identityListToPagerResult(identities))
			pool := NewManagedIdentityPool(client, defaultResourceGroup)
			createdIdentities, err := pool.ClaimIdentities(context.Background(), 10, defaultClusterResourceGroup, defaultClusterName, defaultTimeout)
			Expect(err).To(HaveOccurred())
			Expect(createdIdentities).To(BeEmpty())
			Expect(isIdentityClaimed(*identities[999])).To(BeFalse())
		})
	})

	When("the Identitity pool is empty", func() {
		It("Lists All Identities", func() {
			client.EXPECT().NewListByResourceGroupPager(defaultResourceGroup, gomock.Any()).Return(identityListToPagerResult([]*armmsi.Identity{}))
			pool := NewManagedIdentityPool(client, defaultResourceGroup)
			allIdentities, err := pool.GetAllIdentitiesInPool(context.Background())
			Expect(allIdentities).To(BeEmpty())
			Expect(err).ToNot(HaveOccurred())
		})
		It("Adds and claims new identities", func() {
			client.EXPECT().NewListByResourceGroupPager(defaultResourceGroup, gomock.Any()).Return(identityListToPagerResult([]*armmsi.Identity{}))
			validUntil := time.Now().Add(defaultTimeout).Format(time.RFC3339)

			defaultIdentity := armmsi.Identity{
				Location: &defaultLocation,
				Tags: map[string]*string{
					ClaimedResourceGroupTagKey: &defaultClusterResourceGroup,
					ClaimedClusterTagKey:       &defaultClusterName,
					ClaimedUntilTagKey:         &validUntil,
				},
			}

			identityPool := []*armmsi.Identity{}
			innerCreateOrUpdate := func(ctx context.Context, resourceGroup string, name string, params armmsi.Identity, opts *armmsi.UserAssignedIdentitiesClientCreateOrUpdateOptions) (armmsi.UserAssignedIdentitiesClientCreateOrUpdateResponse, error) {
				newId := armmsi.Identity{
					Name:     &name,
					ID:       to.Ptr(IdentityId(resourceGroup, name)),
					Tags:     params.Tags,
					Location: params.Location,
				}
				identityPool = append(identityPool, &newId)

				return armmsi.UserAssignedIdentitiesClientCreateOrUpdateResponse{
					Identity: newId,
				}, nil
			}
			client.EXPECT().CreateOrUpdate(gomock.Any(), defaultResourceGroup, gomock.Any(), gomock.Eq(defaultIdentity), gomock.Any()).DoAndReturn(innerCreateOrUpdate)
			client.EXPECT().CreateOrUpdate(gomock.Any(), defaultResourceGroup, gomock.Any(), gomock.Eq(defaultIdentity), gomock.Any()).DoAndReturn(innerCreateOrUpdate)
			pool := NewManagedIdentityPool(client, defaultResourceGroup)
			numIdentities := 2
			identities, err := pool.ClaimIdentities(context.Background(), numIdentities, defaultClusterResourceGroup, defaultClusterName, defaultTimeout)
			Expect(err).ToNot(HaveOccurred())
			Expect(identities).To(HaveLen(numIdentities))

			for _, id := range identities {
				Expect(id.Location).To(HaveValue(Equal(defaultLocation)))
				Expect(id.Tags).To(Equal(defaultIdentity.Tags))
			}
		})
	})
})

func TestManagedIdentitiesPool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Managed Identities Pool Suite")
}

func IdentityId(rg string, name string) string {
	return fmt.Sprintf("/subscriptions/12341234/resourcegroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", rg, name)
}

func identityListToPagerResult(identities []*armmsi.Identity) *runtime.Pager[armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse] {
	return runtime.NewPager(runtime.PagingHandler[armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse]{
		More: func(armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse) bool {
			return false
		},
		Fetcher: func(context.Context, *armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse) (armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse, error) {
			return armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse{
				UserAssignedIdentitiesListResult: armmsi.UserAssignedIdentitiesListResult{
					NextLink: to.Ptr(""),
					Value:    identities,
				},
			}, nil
		},
		Tracer: tracing.Tracer{},
	})
}
