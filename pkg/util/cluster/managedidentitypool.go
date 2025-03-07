package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/go-autorest/autorest/azure"

	armmsiclient "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armmsi"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const UserAssignedIdentityType = "Microsoft.ManagedIdentity/userAssignedIdentities"
const ClaimedResourceGroupTagKey = "CLAIMED_BY_RG"
const ClaimedClusterTagKey = "CLAIMED_BY_CLUSTER"
const ClaimedUntilTagKey = "CLAIMED_UNTIL"

type ManagedIdentityPool struct {
	uaiClient         armmsiclient.UserAssignedIdentitiesClient
	resourceGroupName string
}

func NewManagedIdentityPool(uaiClient armmsiclient.UserAssignedIdentitiesClient, resourceGroupName string) *ManagedIdentityPool {
	return &ManagedIdentityPool{
		uaiClient:         uaiClient,
		resourceGroupName: resourceGroupName,
	}
}

func isUserAssignedIdentity(identity armmsi.Identity) bool {
	if identity.Type == nil {
		return false
	}
	return *identity.Type == UserAssignedIdentityType
}

func isIdentityClaimed(identity armmsi.Identity) bool {
	claimedRg, hasClaimedRg := identity.Tags[ClaimedResourceGroupTagKey]
	claimedCluster, hasClaimedCluster := identity.Tags[ClaimedClusterTagKey]
	claimedUntil, hasClaimedUntil := identity.Tags[ClaimedUntilTagKey]

	// if any of these is unset
	if !(hasClaimedRg && hasClaimedCluster && hasClaimedUntil) {
		return false
	}
	// also, if any of them is nil or ""
	if (claimedRg == nil || *claimedRg == "") ||
		(claimedCluster == nil || *claimedCluster == "") ||
		(claimedUntil == nil || *claimedUntil == "") {
		return false
	}

	parsedClaimedUntil, err := time.Parse(time.RFC3339, *claimedUntil)
	if err != nil {
		return false
	}

	return parsedClaimedUntil.After(time.Now())
}

func (pool *ManagedIdentityPool) GetAllIdentitiesInPool(ctx context.Context) ([]*armmsi.Identity, error) {
	listIdentitiesPager := pool.uaiClient.NewListByResourceGroupPager(pool.resourceGroupName, &armmsi.UserAssignedIdentitiesClientListByResourceGroupOptions{})
	allIdentities := []*armmsi.Identity{}

	for listIdentitiesPager.More() {
		listIdentitesPage, err := listIdentitiesPager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("couldn't get identities from rg: %w", err)
		}

		for _, identity := range listIdentitesPage.Value {
			if identity == nil || !isUserAssignedIdentity(*identity) {
				continue
			}
			allIdentities = append(allIdentities, identity)
		}
	}

	return allIdentities, nil
}

func (pool *ManagedIdentityPool) ClaimIdentities(ctx context.Context, desiredNumberOfIdentities int, clusterResourceGroup string, clusterName string, timeout time.Duration) ([]*armmsi.Identity, error) {
	claimedIdentites := []*armmsi.Identity{}

	allIdentities, err := pool.GetAllIdentitiesInPool(ctx)
	if err != nil {
		return nil, err
	}

	for _, identity := range allIdentities {
		if len(claimedIdentites) == desiredNumberOfIdentities {
			return claimedIdentites, nil
		}

		if isIdentityClaimed(*identity) {
			continue
		}
		// try claiming this identity
		updatedIdentity, err := pool.ClaimIdentity(ctx, *identity, clusterResourceGroup, clusterName, timeout)
		if err != nil {
			return claimedIdentites, err
		}
		claimedIdentites = append(claimedIdentites, updatedIdentity)
	}

	remainingIdentities := desiredNumberOfIdentities - len(claimedIdentites)
	for i := 0; i < remainingIdentities; i++ {
		newIdentity, err := pool.CreateAndClaimIdentity(ctx, clusterResourceGroup, clusterName, timeout)
		if err != nil {
			return claimedIdentites, err
		}

		claimedIdentites = append(claimedIdentites, newIdentity)
	}

	return claimedIdentites, nil
}

func (pool *ManagedIdentityPool) CreateAndClaimIdentity(ctx context.Context, clusterResourceGroup string, clusterName string, timeout time.Duration) (*armmsi.Identity, error) {
	validUntil := time.Now().Add(timeout).Format(time.RFC3339)
	identityName := "id-" + uuid.DefaultGenerator.Generate()

	response, err := pool.uaiClient.CreateOrUpdate(ctx, pool.resourceGroupName, identityName, armmsi.Identity{
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			ClaimedClusterTagKey:       &clusterName,
			ClaimedResourceGroupTagKey: &clusterResourceGroup,
			ClaimedUntilTagKey:         &validUntil,
		},
	}, &armmsi.UserAssignedIdentitiesClientCreateOrUpdateOptions{})

	if err != nil {
		return nil, fmt.Errorf("error updating identity tags: %w", err)
	}

	return &response.Identity, nil
}

func (pool *ManagedIdentityPool) ClaimIdentity(ctx context.Context, identity armmsi.Identity, clusterResourceGroup string, clusterName string, timeout time.Duration) (*armmsi.Identity, error) {
	validUntil := time.Now().Add(timeout).Format(time.RFC3339)

	response, err := pool.uaiClient.Update(ctx, pool.resourceGroupName, *identity.Name, armmsi.IdentityUpdate{
		Tags: map[string]*string{
			ClaimedClusterTagKey:       &clusterName,
			ClaimedResourceGroupTagKey: &clusterResourceGroup,
			ClaimedUntilTagKey:         &validUntil,
		},
	}, &armmsi.UserAssignedIdentitiesClientUpdateOptions{})

	if err != nil {
		return nil, fmt.Errorf("error updating identity tags: %w", err)
	}

	return &response.Identity, nil
}

func (pool *ManagedIdentityPool) FreeIdentity(ctx context.Context, identity armmsi.Identity) (*armmsi.Identity, error) {
	idResource, err := azure.ParseResourceID(*identity.ID)
	if err != nil {
		return nil, err
	}

	response, err := pool.uaiClient.Update(ctx, idResource.ResourceGroup, *identity.Name, armmsi.IdentityUpdate{
		Tags: map[string]*string{
			ClaimedClusterTagKey:       to.Ptr(""),
			ClaimedResourceGroupTagKey: to.Ptr(""),
			ClaimedUntilTagKey:         to.Ptr(""),
		},
	}, &armmsi.UserAssignedIdentitiesClientUpdateOptions{})

	if err != nil {
		return nil, fmt.Errorf("error updating identity tags: %w", err)
	}

	return &response.Identity, nil
}

func (pool *ManagedIdentityPool) FreeAllIdentitiesOfCluster(ctx context.Context, clusterResourceGroup string, clusterName string) error {
	identities, err := pool.GetAllIdentitiesInPool(ctx)
	if err != nil {
		return err
	}

	for _, id := range identities {
		if stringPtrEqualsString(id.Tags[ClaimedClusterTagKey], clusterName) &&
			stringPtrEqualsString(id.Tags[ClaimedResourceGroupTagKey], clusterResourceGroup) {
			if _, err := pool.FreeIdentity(ctx, *id); err != nil {
				return err
			}
		}
	}

	return nil
}

func stringPtrEqualsString(p *string, s string) bool {
	if p == nil {
		return false
	}
	return *p == s
}
