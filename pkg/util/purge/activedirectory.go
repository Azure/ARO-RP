package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
)

func (rc *ResourceCleaner) CleanAAD(ctx context.Context) error {
	roleAssignments, err := rc.listRoleAssignmentsForVnets(ctx)
	if err != nil {
		return err
	}

	return rc.cleanAAD(ctx, roleAssignments)
}

func (rc *ResourceCleaner) listRoleAssignmentsForVnets(ctx context.Context) ([]authorization.RoleAssignment, error) {
	roleAssignmentsResourceGroups := []string{"v4-westeurope", "v4-eastus", "v4-australiasoutheast"}
	var filteredRoleAssignments []authorization.RoleAssignment

	for _, rg := range roleAssignmentsResourceGroups {
		rc.log.Debugf("rg: %s", rg)
		vnets, err := rc.vnetscli.List(ctx, rg)
		if err != nil {
			return nil, err
		}

		for _, vnet := range vnets {
			scope := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s", rc.subscriptionID, rg, *vnet.Name)
			roleAssignments, err := rc.roleassignmentcli.ListForScope(ctx, scope, "")
			if err != nil {
				return nil, err
			}

			for _, rs := range roleAssignments {
				if rs.PrincipalType != authorization.ServicePrincipal {
					// work only on application types
					continue
				}

				filteredRoleAssignments = append(filteredRoleAssignments, rs)
			}
		}
	}

	return filteredRoleAssignments, nil
}

// cleanAAD cleans up the active directory of remaining:
// * applications
// * service principals
// * role bindings
//
// Cleanup of the AAD is based on the idea that:
// * application belonging to ARO has the same name as the clusterResourceGroup
// * the resourceGroup already not exists
// * application is linked to on of preexisting vnets in either: v4-eastus, v4-westeurope, v4-australiasoutheast
//
// The simplified diagram of the process is bellow.
//
//
//   +------------------------------+
//   |   Get roleAssignments from:  |
//   |   * v4-eastus                |
//   |   * v4-westeurope            |
//   |   * v4-australiasoutheast    |
//   |                              |
//   |   Used for shared clusters   |
//   +--------------+---------------+
//                  |
//                  |
//     +------------v--------------+
//     |  For each roleAssignment  |
//     +------------+--------------+
//                  |
//                  |
//   +--------------v---------------+              +--------------------------------+
//   | Get servicePrincipal linked  |   no SP      | roleAssignment is dead and can |
//   | to ARO and roleAssignment    +--------------> be deleted safely              |
//   +--------------+---------------+              +--------------------------------+
//                  |
//                  |
// +----------------v-------------------+
// | Get linked application ID assigned |
// | to ARO                             |
// +----------------+-------------------+
//                  |
//                  |
// +----------------v-------------------+
// |  APP have to:                      |
// |  * be only 1                       |
// |  * have valid resourceGroup name,  |
// |    RG with the name existed        |
// |  * identifierURI have to point to: |
// |    https://az.aro.azure.com/       |
// +----------------+-------------------+
//                  |
//                  |
// +----------------v-----------------+  RG exists  +-------------------------+
// | Matching resourceGroup should not+------------->  Do nothing, ARO exists |
// | exists                           |             +-------------------------+
// +----------------+-----------------+
//                  |
//                  |
//     +------------v--------------+
//     | Delete:                   |
//     | * roleAssignment          |
//     | * servicePrincipal        |
//     | * application             |
//     |                           |
//     | Eveything is from dead ARO|
//     |                           |
//     +---------------------------+
//
func (rc *ResourceCleaner) cleanAAD(ctx context.Context, roleAssignments []authorization.RoleAssignment) error {
	reResourceGroup := regexp.MustCompile(`^[-\p{L}\._\(\)\w]+$`)
	// reApp := regexp.MustCompile(`aro-[a-z0-9]{8}`)

	for _, roleAssignment := range roleAssignments {
		rc.log.Debugf("Processing roleassignment: %s\n", *roleAssignment.PrincipalID)
		servicePrincipal, err := rc.serviceprincipalcli.Get(ctx, *roleAssignment.PrincipalID)
		if err != nil {
			if detailedErr, ok := err.(autorest.DetailedError); ok &&
				detailedErr.StatusCode == http.StatusNotFound {
				// no service principal found than the role assignment is dead and can be deleted
				rc.log.Debugf("Deleting dead roleassignment: %s", *roleAssignment.Name)
				if !rc.dryRun {
					_, err := rc.roleassignmentcli.DeleteByID(ctx, *roleAssignment.ID)
					if err != nil {
						return err
					}
				}
				continue
			}
			rc.log.Errorf("cannot get service principal: %s\n", err.Error())
			continue
			// return err
		}
		rc.log.Debugf("getting app: %s ", *servicePrincipal.AppID)

		filter := fmt.Sprintf("appId eq '%s'", *servicePrincipal.AppID)
		apps, err := rc.applicationscli.List(ctx, filter)
		if err != nil {
			rc.log.Errorf("Cannot get appID: %s, %s", *servicePrincipal.AppID, err.Error())
			return err
		}
		if len(apps) != 1 {
			//something went wrong exit
			rc.log.Debugf("More apps: %v", apps)
			return nil
		}

		app := apps[0]
		if len(*app.IdentifierUris) != 1 ||
			!strings.HasPrefix((*app.IdentifierUris)[0], "https://az.aro.azure.com/") {
			// app does not match ARO pattern
			rc.log.Debugf("Skipping app: %s/%v", *app.DisplayName, *app.IdentifierUris)
			continue
		}

		// clusters can already have the same arbitrary name as cluster resource group
		// checking whether the app is linked to existing cluster can lead to false negatives -> cluster can already be gone and app needs to be deleted
		// therefore only check for valid name
		if !reResourceGroup.MatchString(*app.DisplayName) {
			// app name does not match resource group pattern
			continue
		}

		_, err = rc.resourcegroupscli.Get(ctx, *app.DisplayName)
		if err != nil {
			if detailedErr, ok := err.(autorest.DetailedError); ok &&
				detailedErr.StatusCode == http.StatusNotFound {

				// the resource group should be already gone, therefore this app is safe to be deleted
				// no matching resource group it can be deleted
				rc.log.Debugf("Deleting app: %s", *app.DisplayName)
				rc.log.Debugf("Deleting ServicePrincipal: %s", (*servicePrincipal.ServicePrincipalNames)[0])
				rc.log.Debugf("Deleting roleassignment: %s", *roleAssignment.Name)

				if !rc.dryRun {
					_, err := rc.applicationscli.Delete(ctx, *app.AppID)
					if err != nil {
						return err
					}

					_, err = rc.serviceprincipalcli.Delete(ctx, *servicePrincipal.ObjectID)
					if err != nil {
						return err
					}

					_, err = rc.roleassignmentcli.DeleteByID(ctx, *roleAssignment.ID)
					if err != nil {
						return err
					}
				}

			} else {
				return err
			}
		}
	}

	return nil
}
