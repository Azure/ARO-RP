package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/form3tech-oss/jwt-go"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclaim"
)

type ServicePrincipalValidator interface {
	Validate(token *adal.ServicePrincipalToken) error
}

type defaultSPValidator struct{}

func NewServicePrincipalValidator() defaultSPValidator {
	return defaultSPValidator{}
}

func (v defaultSPValidator) Validate(token *adal.ServicePrincipalToken) error {
	p := &jwt.Parser{}
	c := &azureclaim.AzureClaim{}
	_, _, err := p.ParseUnverified(token.OAuthToken(), c)
	if err != nil {
		return err
	}

	for _, role := range c.Roles {
		if role == "Application.ReadWrite.OwnedBy" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "The provided service principal must not have the Application.ReadWrite.OwnedBy permission.")
		}
	}

	return nil
}
