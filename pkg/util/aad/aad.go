package aad

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"
	"time"

	azgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/form3tech-oss/jwt-go"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

type Manager interface {
	GetToken(ctx context.Context, resource string) (*adal.ServicePrincipalToken, error)
	GetServicePrincipalID(ctx context.Context) (string, error)
}

type manager struct {
	log          *logrus.Entry
	azEnv        *azure.Environment
	tenantID     string
	clientID     string
	clientSecret string
}

// NewManager returns aad manager for AAD operations.
func NewManager(log *logrus.Entry, azEnv *azure.Environment, tenantID, clientID, clientSecret string) Manager {
	// manager should be kept abstract enough so we could call it from cluster
	// or RP context.
	return &manager{
		log:          log,
		azEnv:        azEnv,
		tenantID:     tenantID,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (m *manager) GetServicePrincipalID(ctx context.Context) (string, error) {
	var clusterSPObjectID string

	token, err := m.GetToken(ctx, m.azEnv.GraphEndpoint)
	if err != nil {
		return "", err
	}

	spGraphAuthorizer := autorest.NewBearerAuthorizer(token)

	applications := graphrbac.NewApplicationsClient(m.azEnv, m.tenantID, spGraphAuthorizer)

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	// NOTE: Do not override err with the error returned by wait.PollImmediateUntil.
	// Doing this will not propagate the latest error to the user in case when wait exceeds the timeout
	_ = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		var res azgraphrbac.ServicePrincipalObjectResult
		res, err = applications.GetServicePrincipalsIDByAppID(ctx, m.clientID)
		if err != nil {
			if strings.Contains(err.Error(), "Authorization_IdentityNotFound") {
				m.log.Info(err)
				return false, nil
			}
			return false, err
		}

		clusterSPObjectID = *res.Value
		return true, nil
	}, timeoutCtx.Done())

	return clusterSPObjectID, err
}

// GetToken authenticates in the customer's tenant as the cluster service
// principal and returns a token.
func (m *manager) GetToken(ctx context.Context, resource string) (*adal.ServicePrincipalToken, error) {
	conf := auth.ClientCredentialsConfig{
		ClientID:     m.clientID,
		ClientSecret: m.clientSecret,
		TenantID:     m.tenantID,
		Resource:     resource,
		AADEndpoint:  m.azEnv.ActiveDirectoryEndpoint,
	}

	sp, err := conf.ServicePrincipalToken()
	if err != nil {
		return nil, err
	}

	authorizer := refreshable.NewAuthorizer(sp)

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// NOTE: Do not override err with the error returned by
	// wait.PollImmediateUntil. Doing this will not propagate the latest error
	// to the user in case when wait exceeds the timeout
	_ = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		var done bool
		done, err = authorizer.RefreshWithContext(ctx, m.log)
		if err != nil {
			err = api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "The provided service principal credentials are invalid.")
		}
		if !done || err != nil {
			return false, err
		}

		p := &jwt.Parser{}
		claims := jwt.MapClaims{}
		_, _, err = p.ParseUnverified(authorizer.OAuthToken(), claims)
		if err != nil {
			return false, err
		}

		for _, claim := range []string{"altsecid", "oid", "puid"} {
			if _, found := claims[claim]; found {
				return true, nil
			}
		}

		// populate err with a user-facing error that will be visible if we're
		// not successful.
		m.log.Info("token does not contain the required claims")
		err = api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalClaims, "properties.servicePrincipalProfile", "The provided service principal does not give an access token with at least one of the claims 'altsecid', 'oid' or 'puid'.")

		return false, nil
	}, timeoutCtx.Done())
	if err != nil {
		return nil, err
	}

	return sp, nil
}
