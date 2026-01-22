package actuator

import (
	"fmt"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

type azClients struct {
	fpCred *azidentity.ClientCertificateCredential

	loadBalancerClient        *armnetwork.LoadBalancersClient
	resourceSKUsClient        *armcompute.ResourceSKUsClient
	privateLinkServicesClient *armnetwork.PrivateLinkServicesClient
}

func (t *th) setupAzureClients() error {
	if t.az != nil {
		fpCredClusterTenant, err := t.env.FPNewClientCertificateCredential(t.sub.Subscription.Properties.TenantID, nil)

		if err != nil {
			return fmt.Errorf("failure creating fpCredClusterTenant: %w", err)
		}

		t.az = &azClients{fpCred: fpCredClusterTenant}
	}
	return nil
}

func (t *th) LoadBalancersClient() (armnetwork.LoadBalancersClient, error) {
	err := t.setupAzureClients()
	if err != nil {
		return nil, err
	}

	if t.az.loadBalancerClient != nil {
		armLoadBalancersClient, err := armnetwork.NewLoadBalancersClient(t.sub.ID, t.az.fpCred, t.env.Environment().ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating client: %w", err)
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

	if t.az.resourceSKUsClient != nil {
		resourceSKUsClient, err := armcompute.NewResourceSKUsClient(t.sub.ID, t.az.fpCred, t.env.Environment().ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating client: %w", err)
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

	if t.az.privateLinkServicesClient != nil {
		privateLinkServicesClient, err := armnetwork.NewPrivateLinkServicesClient(t.sub.ID, t.az.fpCred, t.env.Environment().ArmClientOptions())
		if err != nil {
			return nil, fmt.Errorf("failure creating client: %w", err)
		}

		t.az.privateLinkServicesClient = &privateLinkServicesClient
	}

	return *t.az.privateLinkServicesClient, nil
}
