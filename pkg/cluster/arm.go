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

func (i *manager) deployARMTemplate(ctx context.Context, rg string, tName string, t *arm.Template, params map[string]interface{}) error {
	i.log.Printf("deploying %s template", tName)

	err := i.deployments.CreateOrUpdateAndWait(ctx, rg, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   t,
			Parameters: params,
			Mode:       mgmtfeatures.Incremental,
		},
	})

	if azureerrors.IsDeploymentActiveError(err) {
		i.log.Printf("waiting for %s template to be deployed", tName)
		err = i.deployments.Wait(ctx, rg, deploymentName)
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
