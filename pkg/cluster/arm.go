package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
)

func (m *manager) deployARMTemplate(ctx context.Context, resourceGroupName string, deploymentName string, template *arm.Template, parameters map[string]interface{}) error {
	m.log.Printf("deploying %s template", deploymentName)
	err := m.deployments.CreateOrUpdateAndWait(ctx, resourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Parameters: parameters,
			Mode:       mgmtfeatures.Incremental,
		},
	})

	if azureerrors.IsDeploymentActiveError(err) {
		m.log.Printf("waiting for %s template to be deployed", deploymentName)
		err = m.deployments.Wait(ctx, resourceGroupName, deploymentName)
	}

	if azureerrors.HasAuthorizationFailedError(err) ||
		azureerrors.HasLinkedAuthorizationFailedError(err) {
		return err
	}

	serviceErr, _ := err.(*azure.ServiceError) // futures return *azure.ServiceError directly

	// CreateOrUpdate() returns a wrapped *azure.ServiceError
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		serviceErr, _ = detailedErr.Original.(*azure.ServiceError)
	}

	if serviceErr != nil {
		b, _ := json.Marshal(serviceErr)

		return &api.CloudError{
			StatusCode: http.StatusBadRequest,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeDeploymentFailed,
				Message: "Deployment failed.",
				Details: []api.CloudErrorBody{
					{
						Message: string(b),
					},
				},
			},
		}
	}

	return err
}
