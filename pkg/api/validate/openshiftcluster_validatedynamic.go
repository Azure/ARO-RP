package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/apparentlymart/go-cidr/cidr"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	utilpermissions "github.com/Azure/ARO-RP/pkg/util/permissions"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

// OpenShiftClusterDynamicValidator is the dynamic validator interface
type OpenShiftClusterDynamicValidator interface {
	Dynamic(context.Context) error
}

// NewOpenShiftClusterDynamicValidator creates a new OpenShiftClusterDynamicValidator
func NewOpenShiftClusterDynamicValidator(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster) (OpenShiftClusterDynamicValidator, error) {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return nil, err
	}

	fpAuthorizer, err := env.FPAuthorizer(oc.Properties.ServicePrincipalProfile.TenantID, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	return &openShiftClusterDynamicValidator{
		log: log,
		env: env,

		oc: oc,

		fpAuthorizer: fpAuthorizer,

		fpPermissions: authorization.NewPermissionsClient(r.SubscriptionID, fpAuthorizer),
	}, nil
}

type azureClaim struct {
	Roles []string `json:"roles,omitempty"`
}

func (*azureClaim) Valid() error {
	return fmt.Errorf("unimplemented")
}

type openShiftClusterDynamicValidator struct {
	log *logrus.Entry
	env env.Interface

	oc *api.OpenShiftCluster

	fpAuthorizer refreshable.Authorizer

	fpPermissions     authorization.PermissionsClient
	spPermissions     authorization.PermissionsClient
	spProviders       features.ProvidersClient
	spUsage           compute.UsageClient
	spVirtualNetworks network.VirtualNetworksClient
}

// Dynamic validates an OpenShift cluster
func (dv *openShiftClusterDynamicValidator) Dynamic(ctx context.Context) error {
	// TODO: Dynamic() should work on an enriched oc (in update), and it
	// currently doesn't.  One sticking point is handling subnet overlap
	// calculations.

	r, err := azure.ParseResourceID(dv.oc.ID)
	if err != nil {
		return err
	}

	spAuthorizer, err := dv.validateServicePrincipalProfile(ctx)
	if err != nil {
		return err
	}

	dv.spPermissions = authorization.NewPermissionsClient(r.SubscriptionID, spAuthorizer)
	dv.spProviders = features.NewProvidersClient(r.SubscriptionID, spAuthorizer)
	dv.spUsage = compute.NewUsageClient(r.SubscriptionID, spAuthorizer)
	dv.spVirtualNetworks = network.NewVirtualNetworksClient(r.SubscriptionID, spAuthorizer)

	vnetID, _, err := subnet.Split(dv.oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	err = dv.validateVnetPermissions(ctx, spAuthorizer, dv.spPermissions, vnetID, &vnetr, api.CloudErrorCodeInvalidServicePrincipalPermissions, "provided service principal")
	if err != nil {
		return err
	}

	err = dv.validateVnetPermissions(ctx, dv.fpAuthorizer, dv.fpPermissions, vnetID, &vnetr, api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
	if err != nil {
		return err
	}

	// Get after validating permissions
	vnet, err := dv.spVirtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	err = dv.validateRouteTablePermissions(ctx, spAuthorizer, dv.spPermissions, &vnet, api.CloudErrorCodeInvalidServicePrincipalPermissions, "provided service principal")
	if err != nil {
		return err
	}

	err = dv.validateRouteTablePermissions(ctx, dv.fpAuthorizer, dv.fpPermissions, &vnet, api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
	if err != nil {
		return err
	}

	err = dv.validateVnet(ctx, &vnet)
	if err != nil {
		return err
	}

	err = dv.validateProviders(ctx)
	if err != nil {
		return err
	}

	if dv.oc.Properties.ProvisioningState == api.ProvisioningStateCreating {
		err = dv.validateQuotas(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dv *openShiftClusterDynamicValidator) validateServicePrincipalProfile(ctx context.Context) (refreshable.Authorizer, error) {
	dv.log.Print("validateServicePrincipalProfile")

	token, err := aad.GetToken(ctx, dv.log, dv.oc, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	p := &jwt.Parser{}
	c := &azureClaim{}
	_, _, err = p.ParseUnverified(token.OAuthToken(), c)
	if err != nil {
		return nil, err
	}

	for _, role := range c.Roles {
		if role == "Application.ReadWrite.OwnedBy" {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "The provided service principal must not have the Application.ReadWrite.OwnedBy permission.")
		}
	}

	return refreshable.NewAuthorizer(token), nil
}

func (dv *openShiftClusterDynamicValidator) validateVnetPermissions(ctx context.Context, authorizer refreshable.Authorizer, client authorization.PermissionsClient, vnetID string, vnetr *azure.Resource, code, typ string) error {
	dv.log.Printf("validateVnetPermissions (%s)", typ)

	err := validateActions(ctx, dv.log, vnetr, []string{
		"Microsoft.Network/virtualNetworks/subnets/join/action",
		"Microsoft.Network/virtualNetworks/subnets/read",
		"Microsoft.Network/virtualNetworks/subnets/write",
	}, authorizer, client)
	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, code, "", "The %s does not have Contributor permission on vnet '%s'.", typ, vnetID)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet '%s' could not be found.", vnetID)
	}
	return err
}

func (dv *openShiftClusterDynamicValidator) validateRouteTablePermissions(ctx context.Context, authorizer refreshable.Authorizer, client authorization.PermissionsClient, vnet *mgmtnetwork.VirtualNetwork, code, typ string) error {
	err := dv.validateRouteTablePermissionsSubnet(ctx, authorizer, client, vnet, dv.oc.Properties.MasterProfile.SubnetID, "properties.masterProfile.subnetId", code, typ)
	if err != nil {
		return err
	}

	return dv.validateRouteTablePermissionsSubnet(ctx, authorizer, client, vnet, dv.oc.Properties.WorkerProfiles[0].SubnetID, `properties.workerProfiles["worker"].subnetId`, code, typ)
}

func (dv *openShiftClusterDynamicValidator) validateRouteTablePermissionsSubnet(ctx context.Context, authorizer refreshable.Authorizer, client authorization.PermissionsClient, vnet *mgmtnetwork.VirtualNetwork, subnetID, path, code, typ string) error {
	dv.log.Printf("validateRouteTablePermissionsSubnet(%s, %s)", typ, path)

	var s *mgmtnetwork.Subnet
	for _, ss := range *vnet.Subnets {
		if strings.EqualFold(*ss.ID, subnetID) {
			s = &ss
			break
		}
	}
	if s == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The subnet '%s' could not be found.", subnetID)
	}

	if s.RouteTable == nil {
		return nil
	}

	rtr, err := azure.ParseResourceID(*s.RouteTable.ID)
	if err != nil {
		return err
	}

	err = validateActions(ctx, dv.log, &rtr, []string{
		"Microsoft.Network/routeTables/join/action",
		"Microsoft.Network/routeTables/read",
		"Microsoft.Network/routeTables/write",
	}, authorizer, client)
	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, code, "", "The %s does not have Contributor permission on route table '%s'.", typ, *s.RouteTable.ID)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedRouteTable, "", "The route table '%s' could not be found.", *s.RouteTable.ID)
	}
	return err
}

func (dv *openShiftClusterDynamicValidator) validateSubnet(ctx context.Context, vnet *mgmtnetwork.VirtualNetwork, path, subnetID string) (*net.IPNet, error) {
	dv.log.Printf("validateSubnet (%s)", path)

	var s *mgmtnetwork.Subnet
	if vnet.Subnets != nil {
		for _, ss := range *vnet.Subnets {
			if strings.EqualFold(*ss.ID, subnetID) {
				s = &ss
				break
			}
		}
	}
	if s == nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided subnet '%s' could not be found.", subnetID)
	}

	if strings.EqualFold(dv.oc.Properties.MasterProfile.SubnetID, subnetID) {
		if s.PrivateLinkServiceNetworkPolicies == nil ||
			!strings.EqualFold(*s.PrivateLinkServiceNetworkPolicies, "Disabled") {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided subnet '%s' is invalid: must have privateLinkServiceNetworkPolicies disabled.", subnetID)
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
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided subnet '%s' is invalid: must have Microsoft.ContainerRegistry serviceEndpoint.", subnetID)
	}

	if dv.oc.Properties.ProvisioningState == api.ProvisioningStateCreating {
		if s.SubnetPropertiesFormat != nil &&
			s.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided subnet '%s' is invalid: must not have a network security group attached.", subnetID)
		}

	} else {
		nsgID, err := subnet.NetworkSecurityGroupID(dv.oc, *s.ID)
		if err != nil {
			return nil, err
		}

		if s.SubnetPropertiesFormat == nil ||
			s.SubnetPropertiesFormat.NetworkSecurityGroup == nil ||
			!strings.EqualFold(*s.SubnetPropertiesFormat.NetworkSecurityGroup.ID, nsgID) {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided subnet '%s' is invalid: must have network security group '%s' attached.", subnetID, nsgID)
		}
	}

	_, net, err := net.ParseCIDR(*s.AddressPrefix)
	if err != nil {
		return nil, err
	}
	{
		ones, _ := net.Mask.Size()
		if ones > 27 {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided subnet '%s' is invalid: must be /27 or larger.", subnetID)
		}
	}

	return net, nil
}

// validateVnet checks that the vnet does not have custom dns servers set
func (dv *openShiftClusterDynamicValidator) validateVnet(ctx context.Context, vnet *mgmtnetwork.VirtualNetwork) error {
	dv.log.Print("validateVnet")

	master, err := dv.validateSubnet(ctx, vnet, "properties.masterProfile.subnetId", dv.oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	worker, err := dv.validateSubnet(ctx, vnet, `properties.workerProfiles["worker"].subnetId`, dv.oc.Properties.WorkerProfiles[0].SubnetID)
	if err != nil {
		return err
	}

	_, pod, err := net.ParseCIDR(dv.oc.Properties.NetworkProfile.PodCIDR)
	if err != nil {
		return err
	}

	_, service, err := net.ParseCIDR(dv.oc.Properties.NetworkProfile.ServiceCIDR)
	if err != nil {
		return err
	}

	err = cidr.VerifyNoOverlap([]*net.IPNet{master, worker, pod, service}, &net.IPNet{IP: net.IPv4zero, Mask: net.IPMask(net.IPv4zero)})
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided CIDRs must not overlap: '%s'.", err)
	}

	if vnet.DhcpOptions != nil &&
		vnet.DhcpOptions.DNSServers != nil &&
		len(*vnet.DhcpOptions.DNSServers) > 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided vnet '%s' is invalid: custom DNS servers are not supported.", *vnet.ID)
	}

	return nil
}

func (dv *openShiftClusterDynamicValidator) validateProviders(ctx context.Context) error {
	dv.log.Print("validateProviders")

	providers, err := dv.spProviders.List(ctx, nil, "")
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

func validateActions(ctx context.Context, log *logrus.Entry, r *azure.Resource, actions []string, authorizer refreshable.Authorizer, client authorization.PermissionsClient) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		permissions, err := client.ListForResource(ctx, r.ResourceGroup, r.Provider, "", r.ResourceType, r.ResourceName)
		if detailedErr, ok := err.(autorest.DetailedError); ok &&
			detailedErr.StatusCode == http.StatusForbidden {
			log.Print(err)
			err = authorizer.RefreshWithContext(ctx)
			if err != nil {
				return false, err
			}
			return false, nil
		}
		if err != nil {
			return false, err
		}

		for _, action := range actions {
			ok, err := utilpermissions.CanDoAction(permissions, action)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}

		return true, nil
	}, timeoutCtx.Done())
}
