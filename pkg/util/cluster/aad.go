package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"time"

	azgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/gofrs/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (c *Cluster) getServicePrincipal(ctx context.Context, appID string) (string, error) {
	// TODO: we are listing here rather than calling
	// i.applications.GetServicePrincipalsIDByAppID() due to some missing
	// permission with our dev/e2e applications
	sps, err := c.serviceprincipals.List(ctx, fmt.Sprintf("appId eq '%s'", appID))
	if err != nil {
		return "", err
	}

	switch len(sps) {
	case 0:
		return "", nil
	case 1:
		return *sps[0].ObjectID, nil
	default:
		return "", fmt.Errorf("%d service principals found for appId %s", len(sps), appID)
	}
}

func (c *Cluster) createApplication(ctx context.Context, displayName string) (string, string, error) {
	password := uuid.Must(uuid.NewV4()).String()

	// example value: https://test.aro.azure.com/11111111-1111-1111-1111-111111111111
	identifierURI := "https://test." + c.env.Environment().AppSuffix + "/" + uuid.Must(uuid.NewV4()).String()

	app, err := c.applications.Create(ctx, azgraphrbac.ApplicationCreateParameters{
		DisplayName:    &displayName,
		IdentifierUris: &[]string{identifierURI},
		PasswordCredentials: &[]azgraphrbac.PasswordCredential{
			{
				EndDate: &date.Time{Time: time.Now().AddDate(1, 0, 0)},
				Value:   &password,
			},
		},
	})
	if err != nil {
		return "", "", err
	}

	return *app.AppID, password, nil
}

func (c *Cluster) createServicePrincipal(ctx context.Context, appID string) (string, error) {
	var sp azgraphrbac.ServicePrincipal
	var err error

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// NOTE: Do not override err with the error returned by
	// wait.PollImmediateUntil. Doing this will not propagate the latest error
	// to the user in case when wait exceeds the timeout
	_ = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		sp, err = c.serviceprincipals.Create(ctx, azgraphrbac.ServicePrincipalCreateParameters{
			AppID: &appID,
		})
		if detailedErr, ok := err.(autorest.DetailedError); ok &&
			detailedErr.StatusCode == http.StatusForbidden {
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
		return "", err
	}

	return *sp.ObjectID, nil
}

func (c *Cluster) deleteApplication(ctx context.Context, appID string) error {
	apps, err := c.applications.List(ctx, fmt.Sprintf("appId eq '%s'", appID))
	if err != nil {
		return err
	}

	switch len(apps) {
	case 0:
		return nil
	case 1:
		c.log.Print("deleting AAD application")
		_, err = c.applications.Delete(ctx, *apps[0].ObjectID)
		return err
	default:
		return fmt.Errorf("%d applications found for appId %s", len(apps), appID)
	}
}
