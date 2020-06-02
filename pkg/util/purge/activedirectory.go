package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/davecgh/go-spew/spew"
)

func (rc *ResourceCleaner) prepareApps(ctx context.Context) error {
	apps, err := rc.applicationscli.List(ctx, "")
	if err != nil {
		return err
	}

	for _, app := range apps {
		rc.appMap[*app.DisplayName] = append(rc.appMap[*app.DisplayName], *app.AppID)
	}

	spew.Dump(rc.appMap)

	return nil
}

// cleanApp cleans app registration attached to the existing resource group
func (rc *ResourceCleaner) cleanAad(ctx context.Context, resourceGroup mgmtfeatures.ResourceGroup) error {
	appIds, ok := rc.appMap[*resourceGroup.Name]
	if !ok {
		rc.log.Infof("App not found: %s", *resourceGroup.Name)
		return nil
	}

	rc.log.Infof("Found apps: %s/%v", *resourceGroup.Name, appIds)

	for _, appID := range appIds {
		sp, err := rc.applicationscli.GetServicePrincipalsIDByAppID(ctx, appID)
		if err != nil {
			return err
		}

		rc.log.Infof("Deleting app: %s/%s", *resourceGroup.Name, appID)
		if !rc.dryRun {
			// _, err = rc.applicationscli.Delete(ctx, appId)
			// if err != nil {
			// 	return err
			// }
		}

		filter := fmt.Sprintf("$filter=principalId eq %s", *sp.Value)
		roleAssignments, err := rc.roleassignmentcli.List(ctx, filter)
		for _, roleAssignment := range roleAssignments {
			rc.log.Infof("Deleting role assignment: %s/%s/%s", *resourceGroup.Name, appID, roleAssignment.RoleDefinitionID)
			if !rc.dryRun {
				// _, err := rc.roleassignmentcli.DeleteByID(ctx, *roleAssignment.ID)
				// if err != nil {
				// 	return err
				// }
			}
		}

	}

	return nil
}
