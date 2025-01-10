package platformworkloadidentity

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	sdkmsi "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armmsi"
)

func GetPlatformWorkloadIdentityIDs(ctx context.Context, identities map[string]api.PlatformWorkloadIdentity, userAssignedIdentitiesClient armmsi.UserAssignedIdentitiesClient) (map[string]api.PlatformWorkloadIdentity, error) {
	updatedIdentities := make(map[string]api.PlatformWorkloadIdentity, len(identities))

	for operatorName, identity := range identities {
		resourceId, err := arm.ParseResourceID(identity.ResourceID)
		if err != nil {
			return nil, fmt.Errorf("platform workload identity '%s' invalid: %w", operatorName, err)
		}

		identityDetails, err := userAssignedIdentitiesClient.Get(ctx, resourceId.ResourceGroupName, resourceId.Name, &sdkmsi.UserAssignedIdentitiesClientGetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error occured when retrieving platform workload identity '%s' details: %w", operatorName, err)
		}

		updatedIdentities[operatorName] = api.PlatformWorkloadIdentity{
			ResourceID: identity.ResourceID,
			ClientID:   *identityDetails.Properties.ClientID,
			ObjectID:   *identityDetails.Properties.PrincipalID,
		}
	}

	return updatedIdentities, nil
}
