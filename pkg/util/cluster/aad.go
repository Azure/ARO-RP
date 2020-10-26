package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	mgmtgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/date"
	uuid "github.com/satori/go.uuid"
)

func (c *Cluster) getServicePrincipal(ctx context.Context, appID string) (string, error) {
	// TODO: we are listing here rather than calling
	// i.applications.GetServicePrincipalsIDByAppID() due to some missing
	// permission with our dev/e2e applications
	sps, err := c.serviceprincipals.List(ctx, fmt.Sprintf("appId eq '%s'", appID))
	if err != nil {
		return "", err
	}

	if len(sps) != 1 {
		return "", fmt.Errorf("%d service principals found for appId %s", len(sps), appID)
	}

	return *sps[0].ObjectID, nil
}

func (c *Cluster) createApplication(ctx context.Context, displayName string) (string, string, error) {
	password := uuid.NewV4().String()

	app, err := c.applications.Create(ctx, mgmtgraphrbac.ApplicationCreateParameters{
		DisplayName:    &displayName,
		IdentifierUris: &[]string{"https://test.aro.azure.com/" + uuid.NewV4().String()},
		PasswordCredentials: &[]mgmtgraphrbac.PasswordCredential{
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
	sp, err := c.serviceprincipals.Create(ctx, mgmtgraphrbac.ServicePrincipalCreateParameters{
		AppID: &appID,
	})
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
