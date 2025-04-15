package instancemetadata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	jwt "github.com/golang-jwt/jwt/v4"

	"github.com/Azure/ARO-RP/pkg/util/azureclaim"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type prod struct {
	instanceMetadata

	do func(*http.Request) (*http.Response, error)
}

func newProd(ctx context.Context) (InstanceMetadata, error) {
	p := &prod{
		do: http.DefaultClient.Do,
	}

	err := p.populateInstanceMetadata()
	if err != nil {
		return nil, err
	}

	err = p.populateTenantIDFromMSI(ctx)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *prod) populateTenantIDFromMSI(ctx context.Context) error {
	options := p.Environment().ManagedIdentityCredentialOptions("")
	msiTokenCredential, err := azidentity.NewManagedIdentityCredential(options)
	if err != nil {
		return err
	}

	tokenRequestOptions := policy.TokenRequestOptions{
		Scopes: []string{p.Environment().ResourceManagerScope},
	}
	token, err := msiTokenCredential.GetToken(ctx, tokenRequestOptions)
	if err != nil {
		return err
	}

	parser := jwt.NewParser()
	c := &azureclaim.AzureClaim{}
	_, _, err = parser.ParseUnverified(token.Token, c)
	if err != nil {
		return fmt.Errorf("the provided service principal is invalid")
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
		AzEnvironment     string `json:"azEnvironment,omitempty"`
	}

	err = json.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return err
	}

	environment, err := azureclient.EnvironmentFromName(m.AzEnvironment)
	if err != nil {
		return err
	}
	p.environment = &environment
	p.subscriptionID = m.SubscriptionID
	p.location = m.Location
	p.resourceGroup = m.ResourceGroupName

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	p.hostname = hostname

	return nil
}
