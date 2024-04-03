package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/davecgh/go-spew/spew"
	"k8s.io/apimachinery/pkg/util/wait"

	utilgraph "github.com/Azure/ARO-RP/pkg/util/graph"
	msgraph_apps "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/applications"
	msgraph_models "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models"
	msgraph_errors "github.com/Azure/ARO-RP/pkg/util/graph/graphsdk/models/odataerrors"
)

func (c *Cluster) getOrCreateClusterServicePrincipal(ctx context.Context, cluster string) (appId string, appSecret string, spId string, err error) {
	var app msgraph_models.Applicationable
	displayName := "aro-" + cluster
	if os.Getenv("E2E_USE_EXISTING_APPLICATION_PREFIX") != "" {
		applicationPrefix := os.Getenv("E2E_USE_EXISTING_APPLICATION_PREFIX")
		if app, err = c.getExistingApplication(ctx, applicationPrefix); err != nil {
			c.log.Error(err)
			return "", "", "", fmt.Errorf("error retrieving existing application: %w", err)
		}

		c.log.Infof("Using existing application: %s", *app.GetDisplayName())
	} else {
		c.log.Infof("Creating new application: %s", displayName)
		if app, err = c.createApplication(ctx, displayName); err != nil {
			c.log.Error(err)
			return "", "", "", fmt.Errorf("error creating new application: %w", err)
		}
	}

	appId = *app.GetAppId()

	c.log.Infof("Adding password credential to application")
	if appSecret, err = c.addPasswordCredentialToApplication(ctx, app, displayName); err != nil {
		c.log.Error(err)
		return "", "", "", fmt.Errorf("error adding password credential to application: %w", err)
	}

	c.log.Infof("Creating service principal for application")
	if spId, err = c.createServicePrincipal(ctx, appId); err != nil {
		c.log.Error(err)
		return "", "", "", fmt.Errorf("error creating service principal: %w", err)
	}

	return
}

func (c *Cluster) createApplication(ctx context.Context, displayName string) (msgraph_models.Applicationable, error) {
	appBody := msgraph_models.NewApplication()
	appBody.SetDisplayName(&displayName)
	appResult, err := c.spGraphClient.Applications().Post(ctx, appBody, nil)
	if err != nil {
		if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
			c.log.Error(spew.Sdump(oDataError.GetErrorEscaped()))
		} else {
			c.log.Error(err)
		}
		return nil, err
	}
	return appResult, nil
}

func (c *Cluster) getExistingApplication(ctx context.Context, prefix string) (msgraph_models.Applicationable, error) {
	getAppOptions := &msgraph_apps.ApplicationsRequestBuilderGetRequestConfiguration{
		QueryParameters: &msgraph_apps.ApplicationsRequestBuilderGetQueryParameters{
			// TODO: figure out how to include a filter for empty `passwordCredentials` array here, to replace the loop below
			Filter: to.StringPtr(fmt.Sprintf("(startswith(displayName, '%s'))", prefix)),
		},
	}

	getAppsResponse, err := c.spGraphClient.Applications().Get(ctx, getAppOptions)
	if err != nil {
		if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
			c.log.Error(spew.Sdump(oDataError.GetErrorEscaped()))
		} else {
			c.log.Error(err)
		}
		return nil, err
	}

	if len(getAppsResponse.GetValue()) == 0 {
		return nil, fmt.Errorf("no applications present that have the specified prefix")
	}

	for _, app := range getAppsResponse.GetValue() {
		if len(app.GetPasswordCredentials()) == 0 {
			return app, nil
		}
	}

	return nil, fmt.Errorf("no applications present that aren't already in use by e2e")
}

func (c *Cluster) addPasswordCredentialToApplication(ctx context.Context, app msgraph_models.Applicationable, displayName string) (string, error) {
	id := app.GetId()
	endDateTime := time.Now().AddDate(1, 0, 0)

	pwCredential := msgraph_models.NewPasswordCredential()
	pwCredential.SetDisplayName(&displayName)
	pwCredential.SetEndDateTime(&endDateTime)

	pwCredentialRequestBody := msgraph_apps.NewItemAddPasswordPostRequestBody()
	pwCredentialRequestBody.SetPasswordCredential(pwCredential)
	// ByApplicationId is confusingly named, but it refers to
	// the application's Object ID, not to the Application ID.
	// https://learn.microsoft.com/en-us/graph/api/application-addpassword?view=graph-rest-1.0&tabs=http#http-request
	pwResult, err := c.spGraphClient.Applications().ByApplicationId(*id).AddPassword().Post(ctx, pwCredentialRequestBody, nil)
	if err != nil {
		if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
			c.log.Error(spew.Sdump(oDataError.GetErrorEscaped()))
		} else {
			c.log.Error(err)
		}
		return "", err
	}

	return *pwResult.GetSecretText(), nil
}

func (c *Cluster) createServicePrincipal(ctx context.Context, appID string) (string, error) {
	var result msgraph_models.ServicePrincipalable
	var err error

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// NOTE: Do not override err with the error returned by
	// wait.PollImmediateUntil. Doing this will not propagate the latest error
	// to the user in case when wait exceeds the timeout
	_ = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
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
	}, timeoutCtx.Done())
	if err != nil {
		if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
			c.log.Error(spew.Sdump(oDataError.GetErrorEscaped()))
		} else {
			c.log.Error(err)
		}
		return "", err
	}

	return *result.GetId(), nil
}

func (c *Cluster) cleanUpApplication(ctx context.Context, appID string) error {
	filter := fmt.Sprintf("appId eq '%s'", appID)
	requestConfiguration := &msgraph_apps.ApplicationsRequestBuilderGetRequestConfiguration{
		QueryParameters: &msgraph_apps.ApplicationsRequestBuilderGetQueryParameters{
			Filter: &filter,
			Select: []string{"id"},
		},
	}
	result, err := c.spGraphClient.Applications().Get(ctx, requestConfiguration)
	if err != nil {
		if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
			c.log.Error(spew.Sdump(oDataError.GetErrorEscaped()))
		} else {
			c.log.Error(err)
		}
		return err
	}

	getAppsResponse := result.GetValue()
	switch len(getAppsResponse) {
	case 0:
		return nil
	case 1:
		existingAppPrefix := os.Getenv("E2E_USE_EXISTING_APPLICATION_PREFIX")
		if existingAppPrefix != "" && strings.HasPrefix(*getAppsResponse[0].GetDisplayName(), existingAppPrefix) {
			c.log.Print("cleaning up cluster resources on application to allow reuse")
			spObjID, err := utilgraph.GetServicePrincipalIDByAppID(ctx, c.spGraphClient, appID)
			if err != nil {
				if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
					c.log.Error(spew.Sdump(oDataError.GetErrorEscaped()))
				} else {
					c.log.Error(err)
				}
				return err
			}
			if err = c.spGraphClient.ServicePrincipals().ByServicePrincipalId(*spObjID).Delete(ctx, nil); err != nil {
				if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
					c.log.Error(spew.Sdump(oDataError.GetErrorEscaped()))
				} else {
					c.log.Error(err)
				}
				return err
			}

			for _, password := range getAppsResponse[0].GetPasswordCredentials() {
				removePasswordRequestBody := msgraph_apps.NewItemRemovePasswordPostRequestBody()
				removePasswordRequestBody.SetKeyId(password.GetKeyId())

				// ByApplicationId is confusingly named, but it refers to
				// the application's Object ID, not to the Application ID.
				// https://learn.microsoft.com/en-us/graph/api/application-delete?view=graph-rest-1.0&tabs=http#http-request
				if err := c.spGraphClient.Applications().ByApplicationId(*getAppsResponse[0].GetId()).RemovePassword().Post(ctx, removePasswordRequestBody, nil); err != nil {
					if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
						c.log.Error(spew.Sdump(oDataError.GetErrorEscaped()))
					} else {
						c.log.Error(err)
					}
					return err
				}
			}

			return nil
		} else {
			c.log.Print("deleting AAD application")
			// ByApplicationId is confusingly named, but it refers to
			// the application's Object ID, not to the Application ID.
			// https://learn.microsoft.com/en-us/graph/api/application-delete?view=graph-rest-1.0&tabs=http#http-request
			if err := c.spGraphClient.Applications().ByApplicationId(*getAppsResponse[0].GetId()).Delete(ctx, nil); err != nil {
				if oDataError, ok := err.(msgraph_errors.ODataErrorable); ok {
					c.log.Error(spew.Sdump(oDataError.GetErrorEscaped()))
				} else {
					c.log.Error(err)
				}
				return err
			}

			return nil
		}
	default:
		return fmt.Errorf("%d applications found for appId %s", len(getAppsResponse), appID)
	}
}
