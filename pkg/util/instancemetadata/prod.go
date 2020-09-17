package instancemetadata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/dgrijalva/jwt-go"
)

type azureClaim struct {
	TenantID string `json:"tid,omitempty"`
}

func (*azureClaim) Valid() error {
	return fmt.Errorf("unimplemented")
}

type ServicePrincipalToken interface {
	RefreshWithContext(context.Context) error
	OAuthToken() string
}

type prod struct {
	instanceMetadata

	do                              func(*http.Request) (*http.Response, error)
	newServicePrincipalTokenFromMSI func(string, string) (ServicePrincipalToken, error)
}

func newProd(ctx context.Context) (InstanceMetadata, error) {
	p := &prod{
		do: http.DefaultClient.Do,
		newServicePrincipalTokenFromMSI: func(msiEndpoint, resource string) (ServicePrincipalToken, error) {
			return adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
		},
	}

	err := p.populateTenantIDFromMSI(ctx)
	if err != nil {
		return nil, err
	}

	err = p.populateInstanceMetadata()
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *prod) populateTenantIDFromMSI(ctx context.Context) error {
	msiEndpoint, err := adal.GetMSIVMEndpoint()
	if err != nil {
		return err
	}

	token, err := p.newServicePrincipalTokenFromMSI(msiEndpoint, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	err = token.RefreshWithContext(ctx)
	if err != nil {
		return err
	}

	parser := &jwt.Parser{}
	c := &azureClaim{}
	_, _, err = parser.ParseUnverified(token.OAuthToken(), c)
	if err != nil {
		return err
	}

	p.tenantID = c.TenantID

	return nil
}

func (p *prod) populateInstanceMetadata() error {
	req, err := http.NewRequest(http.MethodGet, "http://169.254.169.254/metadata/instance/compute?api-version=2019-03-11", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Metadata", "true")

	resp, err := p.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	if strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0] != "application/json" {
		return fmt.Errorf("unexpected content type %q", resp.Header.Get("Content-Type"))
	}

	var m *struct {
		Location          string `json:"location,omitempty"`
		ResourceGroupName string `json:"resourceGroupName,omitempty"`
		SubscriptionID    string `json:"subscriptionId,omitempty"`
	}

	err = json.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return err
	}

	p.subscriptionID = m.SubscriptionID
	p.location = m.Location
	p.resourceGroup = m.ResourceGroupName

	return nil
}
