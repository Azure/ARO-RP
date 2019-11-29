package subnet

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/jim-minter/rp/pkg/api"
)

// TODO: restructure to allow mocking

// Split splits the given subnetID into a vnetID and subnetName
func Split(subnetID string) (string, string, error) {
	parts := strings.Split(subnetID, "/")
	if len(parts) != 11 {
		return "", "", fmt.Errorf("subnet ID %q has incorrect length", subnetID)
	}

	return strings.Join(parts[:len(parts)-2], "/"), parts[len(parts)-1], nil
}

// Get retrieves the linked subnet using the passed service principal
func Get(ctx context.Context, spp *api.ServicePrincipalProfile, subnetID string) (*network.Subnet, error) {
	vnetID, subnetName, err := Split(subnetID)
	if err != nil {
		return nil, err
	}

	r, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return nil, err
	}

	authorizer, err := auth.NewClientCredentialsConfig(spp.ClientID, spp.ClientSecret, spp.TenantID).Authorizer()
	if err != nil {
		return nil, err
	}

	c := network.NewSubnetsClient(r.SubscriptionID)
	c.Authorizer = authorizer

	subnet, err := c.Get(ctx, r.ResourceGroup, r.ResourceName, subnetName, "")
	if err != nil {
		return nil, err
	}

	return &subnet, nil
}

// CreateOrUpdate updates the linked subnet using the passed service principal
func CreateOrUpdate(ctx context.Context, spp *api.ServicePrincipalProfile, subnetID string, subnet *network.Subnet) error {
	vnetID, subnetName, err := Split(subnetID)
	if err != nil {
		return err
	}

	r, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	authorizer, err := auth.NewClientCredentialsConfig(spp.ClientID, spp.ClientSecret, spp.TenantID).Authorizer()
	if err != nil {
		return err
	}

	c := network.NewSubnetsClient(r.SubscriptionID)
	c.Authorizer = authorizer

	future, err := c.CreateOrUpdate(ctx, r.ResourceGroup, r.ResourceName, subnetName, *subnet)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

// NetworkSecurityGroupID returns the NetworkSecurityGroup ID for a given subnet
// ID
func NetworkSecurityGroupID(oc *api.OpenShiftCluster, subnetID string) (string, error) {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return "", err
	}

	switch {
	case strings.EqualFold(subnetID, oc.Properties.MasterProfile.SubnetID):
		return "/subscriptions/" + r.SubscriptionID + "/resourceGroups/" + oc.Properties.ResourceGroup + "/providers/Microsoft.Network/networkSecurityGroups/" + oc.Properties.InfraID + "-controlplane-nsg", nil
	case strings.EqualFold(subnetID, oc.Properties.WorkerProfiles[0].SubnetID):
		return "/subscriptions/" + r.SubscriptionID + "/resourceGroups/" + oc.Properties.ResourceGroup + "/providers/Microsoft.Network/networkSecurityGroups/" + oc.Properties.InfraID + "-node-nsg", nil
	default:
		return "", fmt.Errorf("unknown subnetID %q", subnetID)
	}
}
