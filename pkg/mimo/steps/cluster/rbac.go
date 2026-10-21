package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sdkauthorization "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"

	"github.com/Azure/ARO-RP/pkg/util/mimo"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

// The Contributor assignment on the managed resource group is granted at
// install time and on user-triggered updates, but is not otherwise recoverable
// if a customer removes it.
func EnsureClusterServicePrincipalRBAC(ctx context.Context) error {
	tc, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	oc := tc.GetOpenShiftClusterDocument().OpenShiftCluster

	if oc.UsesWorkloadIdentity() {
		tc.SetResultMessage("cluster uses workload identity and has no cluster service principal")
		return nil
	}

	spp := oc.Properties.ServicePrincipalProfile
	if spp == nil || spp.SPObjectID == "" {
		return mimo.TerminalError(errors.New("cluster service principal object id is not populated"))
	}

	resourceGroupID := oc.Properties.ClusterProfile.ResourceGroupID
	resourceGroup := stringutils.LastTokenByte(resourceGroupID, '/')

	client, err := tc.RoleAssignmentsClient()
	if err != nil {
		return mimo.TerminalError(err)
	}

	assignments, err := client.ListForResourceGroup(ctx, resourceGroup, nil)
	if err != nil {
		return mimo.TransientError(fmt.Errorf("listing role assignments for resource group %s: %w", resourceGroup, err))
	}

	for _, assignment := range assignments {
		if isContributorAssignment(assignment, resourceGroupID, spp.SPObjectID) {
			tc.SetResultMessage("cluster service principal already has Contributor on the managed resource group")
			return nil
		}
	}

	roleDefinitionID := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s", tc.GetSubscriptionDocument().ID, rbac.RoleContributor)

	_, err = client.Create(ctx, resourceGroupID, uuid.DefaultGenerator.Generate(), sdkauthorization.RoleAssignmentCreateParameters{
		Properties: &sdkauthorization.RoleAssignmentProperties{
			RoleDefinitionID: &roleDefinitionID,
			PrincipalID:      &spp.SPObjectID,
			PrincipalType:    new(sdkauthorization.PrincipalTypeServicePrincipal),
		},
	}, nil)
	if err != nil {
		return mimo.TransientError(fmt.Errorf("creating Contributor role assignment for cluster service principal: %w", err))
	}

	tc.SetResultMessage("restored Contributor on the managed resource group for the cluster service principal")
	return nil
}

func isContributorAssignment(assignment *sdkauthorization.RoleAssignment, resourceGroupID, spObjectID string) bool {
	if assignment == nil || assignment.Properties == nil {
		return false
	}

	p := assignment.Properties
	if p.Scope == nil || p.PrincipalID == nil || p.RoleDefinitionID == nil {
		return false
	}

	// ListForResourceGroup returns assignments inherited from the subscription
	// too, so the scope has to be matched explicitly
	return strings.EqualFold(*p.Scope, resourceGroupID) &&
		strings.EqualFold(*p.PrincipalID, spObjectID) &&
		strings.HasSuffix(strings.ToLower(*p.RoleDefinitionID), rbac.RoleContributor)
}
