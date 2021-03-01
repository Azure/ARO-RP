package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/form3tech-oss/jwt-go"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclaim"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/permissions"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

// Dynamic validate in the operator context.
type Dynamic interface {
	ValidateVnetPermissions(ctx context.Context, vnetID string) error
	ValidateRouteTablesPermissions(ctx context.Context, subnetIDs []string) error
	ValidateCIDRRanges(ctx context.Context, subnetIDs []string, additionalCIDRs ...string) error
	ValidateVnetLocation(ctx context.Context, vnetID string, location string) error
	ValidateProviders(ctx context.Context) error
	ValidateClusterServicePrincipalProfile(ctx context.Context, clientID, clientSecret, tenantID string) error

	ValidateQuota(ctx context.Context, oc *api.OpenShiftCluster) error
}

type dynamic struct {
	log            *logrus.Entry
	authorizerType AuthorizerType
	azEnv          *azure.Environment

	permissions     authorization.PermissionsClient
	providers       features.ProvidersClient
	virtualNetworks virtualNetworksGetClient
	spUsage         compute.UsageClient
}

type AuthorizerType string

const AuthorizerFirstParty AuthorizerType = "resource provider"
const AuthorizerClusterServicePrincipal AuthorizerType = "cluster"

func NewValidator(log *logrus.Entry, azEnv *azure.Environment, subscriptionID string, authorizer refreshable.Authorizer, authorizerType AuthorizerType) (*dynamic, error) {
	return &dynamic{
		log:            log,
		authorizerType: authorizerType,
		azEnv:          azEnv,

		permissions:     authorization.NewPermissionsClient(azEnv, subscriptionID, authorizer),
		providers:       features.NewProvidersClient(azEnv, subscriptionID, authorizer),
		spUsage:         compute.NewUsageClient(azEnv, subscriptionID, authorizer),
		virtualNetworks: newVirtualNetworksCache(network.NewVirtualNetworksClient(azEnv, subscriptionID, authorizer)),
	}, nil
}

func (dv *dynamic) ValidateVnetPermissions(ctx context.Context, vnetID string) error {
	dv.log.Printf("ValidateVnetPermissions")
	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	errCode := api.CloudErrorCodeInvalidResourceProviderPermissions
	if dv.authorizerType == AuthorizerClusterServicePrincipal {
		errCode = api.CloudErrorCodeInvalidServicePrincipalPermissions
	}

	err = dv.validateActions(ctx, &vnetr, []string{
		"Microsoft.Network/virtualNetworks/join/action",
		"Microsoft.Network/virtualNetworks/read",
		"Microsoft.Network/virtualNetworks/write",
		"Microsoft.Network/virtualNetworks/subnets/join/action",
		"Microsoft.Network/virtualNetworks/subnets/read",
		"Microsoft.Network/virtualNetworks/subnets/write",
	})

	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, errCode, "", "The %s service principal does not have Network Contributor permission on vnet '%s'.", dv.authorizerType, vnetID)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet '%s' could not be found.", vnetID)
	}
	return err
}

// ValidateRouteValidateRouteTablesPermissions will validate permissions on each provided subnet
// We are not extracting subnets from the vnet because vnet might contain subnets used
// outside ARO. By explicitly asking caller to provide these we make it callers responsibility.
func (dv *dynamic) ValidateRouteTablesPermissions(ctx context.Context, subnetIDs []string) error {
	if len(subnetIDs) == 0 {
		return fmt.Errorf("no subnets provided")
	}

	vnetID, _, err := subnet.Split(subnetIDs[0])

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	vnet, err := dv.virtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	m := map[string]bool{}

	for _, s := range subnetIDs {
		rtID, err := getRouteTableID(&vnet, s)
		if err != nil {
			return err
		}

		if _, ok := m[strings.ToLower(rtID)]; ok || rtID == "" {
			continue
		}
		m[strings.ToLower(rtID)] = true
	}

	rts := make([]string, 0, len(m))
	for rt := range m {
		rts = append(rts, rt)
	}

	for _, rt := range rts {
		err := dv.validateRouteTablePermissions(ctx, rt)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dv *dynamic) validateRouteTablePermissions(ctx context.Context, rtID string) error {
	dv.log.Printf("validateRouteTablePermissions")

	rtr, err := azure.ParseResourceID(rtID)
	if err != nil {
		return err
	}

	errCode := api.CloudErrorCodeInvalidResourceProviderPermissions
	if dv.authorizerType == AuthorizerClusterServicePrincipal {
		errCode = api.CloudErrorCodeInvalidServicePrincipalPermissions
	}

	err = dv.validateActions(ctx, &rtr, []string{
		"Microsoft.Network/routeTables/join/action",
		"Microsoft.Network/routeTables/read",
		"Microsoft.Network/routeTables/write",
	})
	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, errCode, "", "The %s service principal does not have Network Contributor permission on route table '%s'.", dv.authorizerType, rtID)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedRouteTable, "", "The route table '%s' could not be found.", rtID)
	}
	return err
}

func (dv *dynamic) validateActions(ctx context.Context, r *azure.Resource, actions []string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	return wait.PollImmediateUntil(20*time.Second, func() (bool, error) {
		dv.log.Debug("retry validateActions")
		perms, err := dv.permissions.ListForResource(ctx, r.ResourceGroup, r.Provider, "", r.ResourceType, r.ResourceName)

		if detailedErr, ok := err.(autorest.DetailedError); ok &&
			detailedErr.StatusCode == http.StatusForbidden {
			return false, steps.ErrWantRefresh
		}
		if err != nil {
			return false, err
		}

		for _, action := range actions {
			ok, err := permissions.CanDoAction(perms, action)
			if !ok || err != nil {
				// TODO(jminter): I don't understand if there are genuinely
				// cases where CanDoAction can return false then true shortly
				// after. I'm a little skeptical; if it can't happen we can
				// simplify this code.  We should add a metric on this.
				return false, err
			}
		}

		return true, nil
	}, timeoutCtx.Done())
}

func (dv *dynamic) ValidateCIDRRanges(ctx context.Context, subnetIDs []string, additionalCIDRs ...string) error {
	dv.log.Print("ValidateCIDRRanges")

	vnetID, _, err := subnet.Split(subnetIDs[0])
	if err != nil {
		return err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	vnet, err := dv.virtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	var CIDRArray []*net.IPNet

	// unique names of subnets from all node pools
	for _, subnet := range subnetIDs {
		s := findSubnet(&vnet, subnet)
		if s != nil {
			_, net, err := net.ParseCIDR(*s.AddressPrefix)
			if err != nil {
				return err
			}
			CIDRArray = append(CIDRArray, net)
		}
	}

	for _, c := range additionalCIDRs {
		_, cidr, err := net.ParseCIDR(c)
		if err != nil {
			return err
		}
		CIDRArray = append(CIDRArray, cidr)
	}

	err = cidr.VerifyNoOverlap(CIDRArray, &net.IPNet{IP: net.IPv4zero, Mask: net.IPMask(net.IPv4zero)})
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided CIDRs must not overlap: '%s'.", err)
	}

	return nil
}

func (dv *dynamic) ValidateVnetLocation(ctx context.Context, vnetID string, location string) error {
	dv.log.Print("ValidateVnetLocation")
	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	vnet, err := dv.virtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	if !strings.EqualFold(*vnet.Location, location) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet location '%s' must match the cluster location '%s'.", *vnet.Location, location)
	}

	return nil
}

func (dv *dynamic) validateSubnets(ctx context.Context, oc *api.OpenShiftCluster, subnetIDs []string) error {
	dv.log.Printf("validateSubnet")
	if len(subnetIDs) == 0 {
		return fmt.Errorf("no subnets found")
	}

	vnetID, _, err := subnet.Split(subnetIDs[0])
	if err != nil {
		return err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	vnet, err := dv.virtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, "")
	if err != nil {
		return err
	}
	for _, subnetID := range subnetIDs {
		s := findSubnet(&vnet, subnetID)
		if s == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided subnet '%s' could not be found.", subnetID)
		}

		if strings.EqualFold(oc.Properties.MasterProfile.SubnetID, subnetID) {
			if s.PrivateLinkServiceNetworkPolicies == nil ||
				!strings.EqualFold(*s.PrivateLinkServiceNetworkPolicies, "Disabled") {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided subnet '%s' is invalid: must have privateLinkServiceNetworkPolicies disabled.", subnetID)
			}
		}

		var found bool
		if s.ServiceEndpoints != nil {
			for _, se := range *s.ServiceEndpoints {
				if strings.EqualFold(*se.Service, "Microsoft.ContainerRegistry") &&
					se.ProvisioningState == mgmtnetwork.Succeeded {
					found = true
					break
				}
			}
		}
		if !found {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided subnet '%s' is invalid: must have Microsoft.ContainerRegistry serviceEndpoint.", subnetID)
		}

		if oc.Properties.ProvisioningState == api.ProvisioningStateCreating {
			if s.SubnetPropertiesFormat != nil &&
				s.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided subnet '%s' is invalid: must not have a network security group attached.", subnetID)
			}

		} else {
			nsgID, err := subnet.NetworkSecurityGroupID(oc, *s.ID)
			if err != nil {
				return err
			}

			if s.SubnetPropertiesFormat == nil ||
				s.SubnetPropertiesFormat.NetworkSecurityGroup == nil ||
				!strings.EqualFold(*s.SubnetPropertiesFormat.NetworkSecurityGroup.ID, nsgID) {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided subnet '%s' is invalid: must have network security group '%s' attached.", subnetID, nsgID)
			}
		}

		_, net, err := net.ParseCIDR(*s.AddressPrefix)
		if err != nil {
			return err
		}

		ones, _ := net.Mask.Size()
		if ones > 27 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided subnet '%s' is invalid: must be /27 or larger.", subnetID)
		}
	}

	return nil
}

func (dv *dynamic) ValidateProviders(ctx context.Context) error {
	dv.log.Print("ValidateProviders")

	providers, err := dv.providers.List(ctx, nil, "")
	if err != nil {
		return err
	}

	providerMap := make(map[string]mgmtfeatures.Provider, len(providers))

	for _, provider := range providers {
		providerMap[*provider.Namespace] = provider
	}

	for _, provider := range []string{
		"Microsoft.Authorization",
		"Microsoft.Compute",
		"Microsoft.Network",
		"Microsoft.Storage",
	} {
		if providerMap[provider].RegistrationState == nil ||
			*providerMap[provider].RegistrationState != "Registered" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorResourceProviderNotRegistered, "", "The resource provider '%s' is not registered.", provider)
		}
	}

	return nil
}

func (dv *dynamic) ValidateClusterServicePrincipalProfile(ctx context.Context, clientID, clientSecret, tenantID string) error {
	// TODO: once aad.GetToken is mockable, write a unit test for this function
	log.Print("ValidateClusterServicePrincipalProfile")

	token, err := aad.GetToken(ctx, dv.log, clientID, clientSecret, tenantID, dv.azEnv.ActiveDirectoryEndpoint, dv.azEnv.GraphEndpoint)
	if err != nil {
		return err
	}

	p := &jwt.Parser{}
	c := &azureclaim.AzureClaim{}
	_, _, err = p.ParseUnverified(token.OAuthToken(), c)
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

func getRouteTableID(vnet *mgmtnetwork.VirtualNetwork, subnetID string) (string, error) {
	s := findSubnet(vnet, subnetID)
	if s == nil {
		return "", api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The subnet '%s' could not be found.", subnetID)
	}

	if s.RouteTable == nil {
		return "", nil
	}

	return *s.RouteTable.ID, nil
}

func findSubnet(vnet *mgmtnetwork.VirtualNetwork, subnetID string) *mgmtnetwork.Subnet {
	if vnet.Subnets != nil {
		for _, s := range *vnet.Subnets {
			if strings.EqualFold(*s.ID, subnetID) {
				return &s
			}
		}
	}

	return nil
}
