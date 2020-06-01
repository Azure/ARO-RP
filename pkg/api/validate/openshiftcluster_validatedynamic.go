package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net"
	"net/http"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
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

		oc:          oc,
		spValidator: NewServicePrincipalValidator(log, &oc.Properties.ServicePrincipalProfile, oc.ID, oc.Properties.MasterProfile.SubnetID, oc.Properties.WorkerProfiles[0].SubnetID),

		fpAuthorizer: fpAuthorizer,

		fpPermissions: authorization.NewPermissionsClient(r.SubscriptionID, fpAuthorizer),
	}, nil
}

type openShiftClusterDynamicValidator struct {
	log *logrus.Entry
	env env.Interface

	oc *api.OpenShiftCluster

	fpAuthorizer refreshable.Authorizer

	fpPermissions     authorization.PermissionsClient
	spProviders       features.ProvidersClient
	spUsage           compute.UsageClient
	spVirtualNetworks network.VirtualNetworksClient

	spValidator ServicePrincipalValidator
}

// Dynamic validates an OpenShift cluster
func (dv *openShiftClusterDynamicValidator) Dynamic(ctx context.Context) error {
	// TODO: Dynamic() should work on an enriched oc (in update), and it
	// currently doesn't.  One sticking point is handling subnet overlap
	// calculations.

	err := dv.spValidator.Validate(ctx)
	if err != nil {
		return err
	}

	r, err := azure.ParseResourceID(dv.oc.ID)
	if err != nil {
		return err
	}

	dv.spProviders = features.NewProvidersClient(r.SubscriptionID, dv.spValidator.Authorizer())
	dv.spUsage = compute.NewUsageClient(r.SubscriptionID, dv.spValidator.Authorizer())
	dv.spVirtualNetworks = network.NewVirtualNetworksClient(r.SubscriptionID, dv.spValidator.Authorizer())

	vnetID, _, err := subnet.Split(dv.oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	err = validateVnetPermissions(ctx, dv.log, dv.fpAuthorizer, dv.fpPermissions, vnetID, &vnetr, api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
	if err != nil {
		return err
	}

	// Get after validating permissions
	vnet, err := dv.spVirtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, "")
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

func (dv *openShiftClusterDynamicValidator) validateRouteTablePermissions(ctx context.Context, authorizer refreshable.Authorizer, client authorization.PermissionsClient, vnet *mgmtnetwork.VirtualNetwork, code, typ string) error {
	err := validateRouteTablePermissionsSubnet(ctx, dv.log, authorizer, client, vnet, dv.oc.Properties.MasterProfile.SubnetID, "properties.masterProfile.subnetId", code, typ)
	if err != nil {
		return err
	}

	return validateRouteTablePermissionsSubnet(ctx, dv.log, authorizer, client, vnet, dv.oc.Properties.WorkerProfiles[0].SubnetID, `properties.workerProfiles["worker"].subnetId`, code, typ)
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
