package subnet

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/jim-minter/rp/pkg/api"
)

type Manager interface {
	Get(ctx context.Context, subnetID string) (*network.Subnet, error)
	CreateOrUpdate(ctx context.Context, subnetID string, subnet *network.Subnet) error
}

type manager struct {
	subnets network.SubnetsClient
}

func NewManager(subscriptionID string, spAuthorizer autorest.Authorizer) Manager {
	m := &manager{
		subnets: network.NewSubnetsClient(subscriptionID),
	}

	m.subnets.Authorizer = spAuthorizer

	return m
}

// Get retrieves the linked subnet
func (m *manager) Get(ctx context.Context, subnetID string) (*network.Subnet, error) {
	vnetID, subnetName, err := Split(subnetID)
	if err != nil {
		return nil, err
	}

	r, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return nil, err
	}

	subnet, err := m.subnets.Get(ctx, r.ResourceGroup, r.ResourceName, subnetName, "")
	if err != nil {
		return nil, err
	}

	return &subnet, nil
}

// CreateOrUpdate updates the linked subnet
func (m *manager) CreateOrUpdate(ctx context.Context, subnetID string, subnet *network.Subnet) error {
	vnetID, subnetName, err := Split(subnetID)
	if err != nil {
		return err
	}

	r, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	future, err := m.subnets.CreateOrUpdate(ctx, r.ResourceGroup, r.ResourceName, subnetName, *subnet)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, m.subnets.Client)
}

// Split splits the given subnetID into a vnetID and subnetName
func Split(subnetID string) (string, string, error) {
	parts := strings.Split(subnetID, "/")
	if len(parts) != 11 {
		return "", "", fmt.Errorf("subnet ID %q has incorrect length", subnetID)
	}

	return strings.Join(parts[:len(parts)-2], "/"), parts[len(parts)-1], nil
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
		return "/subscriptions/" + r.SubscriptionID + "/resourceGroups/" + oc.Properties.ResourceGroup + "/providers/Microsoft.Network/networkSecurityGroups/aro-controlplane-nsg", nil
	case strings.EqualFold(subnetID, oc.Properties.WorkerProfiles[0].SubnetID):
		return "/subscriptions/" + r.SubscriptionID + "/resourceGroups/" + oc.Properties.ResourceGroup + "/providers/Microsoft.Network/networkSecurityGroups/aro-node-nsg", nil
	default:
		return "", fmt.Errorf("unknown subnetID %q", subnetID)
	}
}
