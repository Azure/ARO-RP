package template

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/resources"
)

func templateDeploy(ctx context.Context, log *logrus.Entry, deployments resources.DeploymentsClient, t *arm.Template, parameters map[string]interface{}, resourceGroup string) error {
	log.Print("deploying resources template")
	err := deployments.CreateOrUpdateAndWait(ctx, resourceGroup, "azuredeploy", mgmtresources.Deployment{
		Properties: &mgmtresources.DeploymentProperties{
			Template:   t,
			Parameters: parameters,
			Mode:       mgmtresources.Incremental,
		},
	})
	if err != nil {
		if detailedErr, ok := err.(autorest.DetailedError); ok {
			if requestErr, ok := detailedErr.Original.(azure.RequestError); ok &&
				requestErr.ServiceError != nil &&
				requestErr.ServiceError.Code == "DeploymentActive" {
				log.Print("waiting for template")
				err = deployments.Wait(ctx, resourceGroup, "azuredeploy")
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}
