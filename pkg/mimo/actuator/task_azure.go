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

type azClients struct {
	fpCredClusterTenant azcore.TokenCredential
	fpCredRPTenant      azcore.TokenCredential

	// Store these as pointers to interfaces so that nil values make sense, as
	// interfaces with a nil value are a pain to determine
	interfacesClient          *armnetwork.InterfacesClient
	loadBalancerClient        *armnetwork.LoadBalancersClient
	resourceSKUsClient        *armcompute.ResourceSKUsClient
	privateLinkServicesClient *armnetwork.PrivateLinkServicesClient
	registriesClient          *armcontainerregistry.RegistriesClient
	tokensClient              *armcontainerregistry.TokensClient
}

func (t *th) setupAzureClients() error {
	if t.az == nil {
		if t.sub == nil || t.sub.Subscription == nil || t.sub.Subscription.Properties == nil || t.sub.Subscription.Properties.TenantID == "" {
			return errInvalidSubDoc
		}

		fpCredClusterTenant, err := t.env.FPNewClientCertificateCredential(t.sub.Subscription.Properties.TenantID, nil)
		if err != nil {
			return fmt.Errorf("%w: %w", errCreatingFpCredClusterTenant, err)
		}

		fpCredRPTenant, err := t.env.FPNewClientCertificateCredential(t.env.TenantID(), nil)
		if err != nil {
			return fmt.Errorf("%w: %w", errCreatingFpCredRPTenant, err)
		}

		t.az = &azClients{
			fpCredClusterTenant: fpCredClusterTenant,
			fpCredRPTenant:      fpCredRPTenant,
		}
	}
	return nil
}

func (t *th) LoadBalancersClient() (armnetwork.LoadBalancersClient, error) {
	err := t.setupAzureClients()
	if err != nil {
		return nil, err
	}

	if t.az.loadBalancerClient == nil {
		armLoadBalancersClient, err := armnetwork.NewLoadBalancersClient(t.sub.ID, t.az.fpCredClusterTenant, t.env.ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating LoadBalancersClient: %w", err)
		}

		t.az.loadBalancerClient = &armLoadBalancersClient
	}

	return *t.az.loadBalancerClient, nil
}

func (t *th) ResourceSKUsClient() (armcompute.ResourceSKUsClient, error) {
	err := t.setupAzureClients()
	if err != nil {
		return nil, err
	}

	if t.az.resourceSKUsClient == nil {
		resourceSKUsClient, err := armcompute.NewResourceSKUsClient(t.sub.ID, t.az.fpCredClusterTenant, t.env.ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating ResourceSKUsClient: %w", err)
		}

		t.az.resourceSKUsClient = &resourceSKUsClient
	}

	return *t.az.resourceSKUsClient, nil
}

func (t *th) PrivateLinkServicesClient() (armnetwork.PrivateLinkServicesClient, error) {
	err := t.setupAzureClients()
	if err != nil {
		return nil, err
	}

	if t.az.privateLinkServicesClient == nil {
		privateLinkServicesClient, err := armnetwork.NewPrivateLinkServicesClient(t.sub.ID, t.az.fpCredClusterTenant, t.env.ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating PrivateLinkServicesClient: %w", err)
		}

		t.az.privateLinkServicesClient = &privateLinkServicesClient
	}

	return *t.az.privateLinkServicesClient, nil
}

func (t *th) InterfacesClient() (armnetwork.InterfacesClient, error) {
	err := t.setupAzureClients()
	if err != nil {
		return nil, err
	}

	if t.az.interfacesClient == nil {
		interfacesClient, err := armnetwork.NewInterfacesClient(t.sub.ID, t.az.fpCredClusterTenant, t.env.ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating InterfacesClient: %w", err)
		}

		t.az.interfacesClient = &interfacesClient
	}

	return *t.az.interfacesClient, nil
}

func (t *th) FirstPartyTokensClient() (armcontainerregistry.TokensClient, error) {
	err := t.setupAzureClients()
	if err != nil {
		return nil, err
	}

	acrR, err := azure.ParseResourceID(t.env.ACRResourceID())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errParsingACRResourceID, err)
	}

	if t.az.tokensClient == nil {
		tokensClient, err := armcontainerregistry.NewTokensClient(acrR.SubscriptionID, t.az.fpCredRPTenant, t.env.ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating TokensClient: %w", err)
		}

		t.az.tokensClient = &tokensClient
	}

	return *t.az.tokensClient, nil
}

func (t *th) FirstPartyRegistriesClient() (armcontainerregistry.RegistriesClient, error) {
	err := t.setupAzureClients()
	if err != nil {
		return nil, err
	}

	acrR, err := azure.ParseResourceID(t.env.ACRResourceID())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errParsingACRResourceID, err)
	}

	if t.az.registriesClient == nil {
		registriesClient, err := armcontainerregistry.NewRegistriesClient(acrR.SubscriptionID, t.az.fpCredRPTenant, t.env.ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating RegistriesClient: %w", err)
		}

		t.az.registriesClient = &registriesClient
	}

	return *t.az.registriesClient, nil
}
