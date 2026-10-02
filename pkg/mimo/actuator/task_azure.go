package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcontainerregistry"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
)

var (
	errInvalidSubDoc               = errors.New("invalid/nil subscription document")
	errCreatingFpCredClusterTenant = errors.New("failure creating fpCredClusterTenant")
	errCreatingFpCredRPTenant      = errors.New("failure creating fpCredRPTenant")
	errParsingACRResourceID        = errors.New("failure parsing ACR ResourceID")
)

type azClusterTenantClients struct {
	fpCred azcore.TokenCredential

	// Store these as pointers to interfaces so that nil values make sense, as
	// interfaces with a nil value are a pain to determine
	interfacesClient          *armnetwork.InterfacesClient
	loadBalancerClient        *armnetwork.LoadBalancersClient
	resourceSKUsClient        *armcompute.ResourceSKUsClient
	privateLinkServicesClient *armnetwork.PrivateLinkServicesClient
}

type azRPTenantClients struct {
	fpCred azcore.TokenCredential

	// Store these as pointers to interfaces so that nil values make sense, as
	// interfaces with a nil value are a pain to determine
	registriesClient *armcontainerregistry.RegistriesClient
	tokensClient     *armcontainerregistry.TokensClient
}

func (t *th) setupClusterAzureClients() error {
	if t.azClusterTenant == nil {
		if t.sub == nil || t.sub.Subscription == nil || t.sub.Subscription.Properties == nil || t.sub.Subscription.Properties.TenantID == "" {
			return errInvalidSubDoc
		}

		fpCredClusterTenant, err := t.env.FPNewClientCertificateCredential(t.sub.Subscription.Properties.TenantID, nil)
		if err != nil {
			return fmt.Errorf("%w: %w", errCreatingFpCredClusterTenant, err)
		}

		t.azClusterTenant = &azClusterTenantClients{
			fpCred: fpCredClusterTenant,
		}
	}
	return nil
}

func (t *th) setupRPTenantAzureClients() error {
	if t.azRPTenant == nil {
		fpCredRPTenant, err := t.env.FPNewClientCertificateCredential(t.env.TenantID(), nil)
		if err != nil {
			return fmt.Errorf("%w: %w", errCreatingFpCredRPTenant, err)
		}

		t.azRPTenant = &azRPTenantClients{
			fpCred: fpCredRPTenant,
		}
	}
	return nil
}

func (t *th) LoadBalancersClient() (armnetwork.LoadBalancersClient, error) {
	err := t.setupClusterAzureClients()
	if err != nil {
		return nil, err
	}

	if t.azClusterTenant.loadBalancerClient == nil {
		armLoadBalancersClient, err := armnetwork.NewLoadBalancersClient(t.sub.ID, t.azClusterTenant.fpCred, t.env.ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating LoadBalancersClient: %w", err)
		}

		t.azClusterTenant.loadBalancerClient = &armLoadBalancersClient
	}

	return *t.azClusterTenant.loadBalancerClient, nil
}

func (t *th) ResourceSKUsClient() (armcompute.ResourceSKUsClient, error) {
	err := t.setupClusterAzureClients()
	if err != nil {
		return nil, err
	}

	if t.azClusterTenant.resourceSKUsClient == nil {
		resourceSKUsClient, err := armcompute.NewResourceSKUsClient(t.sub.ID, t.azClusterTenant.fpCred, t.env.ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating ResourceSKUsClient: %w", err)
		}

		t.azClusterTenant.resourceSKUsClient = &resourceSKUsClient
	}

	return *t.azClusterTenant.resourceSKUsClient, nil
}

func (t *th) PrivateLinkServicesClient() (armnetwork.PrivateLinkServicesClient, error) {
	err := t.setupClusterAzureClients()
	if err != nil {
		return nil, err
	}

	if t.azClusterTenant.privateLinkServicesClient == nil {
		privateLinkServicesClient, err := armnetwork.NewPrivateLinkServicesClient(t.sub.ID, t.azClusterTenant.fpCred, t.env.ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating PrivateLinkServicesClient: %w", err)
		}

		t.azClusterTenant.privateLinkServicesClient = &privateLinkServicesClient
	}

	return *t.azClusterTenant.privateLinkServicesClient, nil
}

func (t *th) InterfacesClient() (armnetwork.InterfacesClient, error) {
	err := t.setupClusterAzureClients()
	if err != nil {
		return nil, err
	}

	if t.azClusterTenant.interfacesClient == nil {
		interfacesClient, err := armnetwork.NewInterfacesClient(t.sub.ID, t.azClusterTenant.fpCred, t.env.ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating InterfacesClient: %w", err)
		}

		t.azClusterTenant.interfacesClient = &interfacesClient
	}

	return *t.azClusterTenant.interfacesClient, nil
}

func (t *th) FirstPartyTokensClient() (armcontainerregistry.TokensClient, error) {
	err := t.setupRPTenantAzureClients()
	if err != nil {
		return nil, err
	}

	acrR, err := azure.ParseResourceID(t.env.ACRResourceID())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errParsingACRResourceID, err)
	}

	if t.azRPTenant.tokensClient == nil {
		tokensClient, err := armcontainerregistry.NewTokensClient(acrR.SubscriptionID, t.azRPTenant.fpCred, t.env.ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating TokensClient: %w", err)
		}

		t.azRPTenant.tokensClient = &tokensClient
	}

	return *t.azRPTenant.tokensClient, nil
}

func (t *th) FirstPartyRegistriesClient() (armcontainerregistry.RegistriesClient, error) {
	err := t.setupRPTenantAzureClients()
	if err != nil {
		return nil, err
	}

	acrR, err := azure.ParseResourceID(t.env.ACRResourceID())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errParsingACRResourceID, err)
	}

	if t.azRPTenant.registriesClient == nil {
		registriesClient, err := armcontainerregistry.NewRegistriesClient(acrR.SubscriptionID, t.azRPTenant.fpCred, t.env.ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating RegistriesClient: %w", err)
		}

		t.azRPTenant.registriesClient = &registriesClient
	}

	return *t.azRPTenant.registriesClient, nil
}
