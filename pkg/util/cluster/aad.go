package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	msgraph_apps "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/applications"
	msgraph_models "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models"
	msgraph_errors "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models/odataerrors"
)

func (c *Cluster) createApplication(ctx context.Context, displayName string) (string, string, error) {
	appBody := msgraph_models.NewApplication()
	appBody.SetDisplayName(&displayName)
	appResult, err := c.spGraphClient.Applications().Post(ctx, appBody, nil)
	if err != nil {
		return "", "", err
	}

	id := *appResult.GetId()
	endDateTime := time.Now().AddDate(1, 0, 0)

	pwCredential := msgraph_models.NewPasswordCredential()
	pwCredential.SetDisplayName(&displayName)
	pwCredential.SetEndDateTime(&endDateTime)

	pwCredentialRequestBody := msgraph_apps.NewItemAddPasswordPostRequestBody()
	pwCredentialRequestBody.SetPasswordCredential(pwCredential)
	// ByApplicationId is confusingly named, but it refers to
	// the application's Object ID, not to the Application ID.
	// https://learn.microsoft.com/en-us/graph/api/application-addpassword?view=graph-rest-1.0&tabs=http#http-request
	pwResult, err := c.spGraphClient.Applications().ByApplicationId(id).AddPassword().Post(ctx, pwCredentialRequestBody, nil)
	if err != nil {
		return "", "", err
	}

	return *appResult.GetAppId(), *pwResult.GetSecretText(), nil
}

func (c *Cluster) createServicePrincipal(ctx context.Context, appID string) (string, error) {
	var result msgraph_models.ServicePrincipalable
	var err error

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// NOTE: Do not override err with the error returned by
	// wait.PollUntilContextCancel. Doing this will not propagate the latest error
	// to the user in case when wait exceeds the timeout
	_ = wait.PollUntilContextCancel(timeoutCtx, 10*time.Second, true, func(ctx context.Context) (bool, error) {
		requestBody := msgraph_models.NewServicePrincipal()
		requestBody.SetAppId(&appID)
		result, err = c.spGraphClient.ServicePrincipals().Post(ctx, requestBody, nil)

		if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok &&
			*oDataError.GetErrorEscaped().GetCode() == "accessDenied" {
			// goal is to retry the following error:
			// graphrbac.ServicePrincipalsClient#Create: Failure responding to
			// request: StatusCode=403 -- Original Error: autorest/azure:
			// Service returned an error. Status=403 Code="Unknown"
			// Message="Unknown service error"
			// Details=[{"odata.error":{"code":"Authorization_RequestDenied","date":"yyyy-mm-ddThh:mm:ss","message":{"lang":"en","value":"When
			// using this permission, the backing application of the service
			// principal being created must in the local
			// tenant"},"requestId":"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"}}]

			return false, nil
		}
		return err == nil, err
	})
	if err != nil {
		return "", err
	}

	return *result.GetId(), nil
}

func (c *Cluster) deleteApplication(ctx context.Context, appID string) error {
	filter := fmt.Sprintf("appId eq '%s'", appID)
	requestConfiguration := &msgraph_apps.ApplicationsRequestBuilderGetRequestConfiguration{
		QueryParameters: &msgraph_apps.ApplicationsRequestBuilderGetQueryParameters{
			Filter: &filter,
			Select: []string{"id"},
		},
	}
	result, err := c.spGraphClient.Applications().Get(ctx, requestConfiguration)
	if err != nil {
		return err
	}

	apps := result.GetValue()
	switch len(apps) {
	case 0:
		return nil
	case 1:
		c.log.Print("deleting AAD application")
		// ByApplicationId is confusingly named, but it refers to
		// the application's Object ID, not to the Application ID.
		// https://learn.microsoft.com/en-us/graph/api/application-delete?view=graph-rest-1.0&tabs=http#http-request
		return c.spGraphClient.Applications().ByApplicationId(*apps[0].GetId()).Delete(ctx, nil)
	default:
		return fmt.Errorf("%d applications found for appId %s", len(apps), appID)
	}
}
