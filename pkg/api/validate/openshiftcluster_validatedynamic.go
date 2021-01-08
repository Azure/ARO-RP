package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/apparentlymart/go-cidr/cidr"
	jwt "github.com/form3tech-oss/jwt-go"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

// OpenShiftClusterDynamicValidator is the dynamic validator interface
type OpenShiftClusterDynamicValidator interface {
	Dynamic(context.Context) error
}

// NewOpenShiftClusterDynamicValidator creates a new OpenShiftClusterDynamicValidator
func NewOpenShiftClusterDynamicValidator(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, subscriptionDoc *api.SubscriptionDocument, fpAuthorizer refreshable.Authorizer) OpenShiftClusterDynamicValidator {
	return &openShiftClusterDynamicValidator{
		log: log,
		env: env,

		oc:              oc,
		subscriptionDoc: subscriptionDoc,
		fpAuthorizer:    fpAuthorizer,
		providers:       features.NewProvidersClient(env.Environment(), subscriptionDoc.ID, fpAuthorizer),
	}
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

	oc              *api.OpenShiftCluster
	subscriptionDoc *api.SubscriptionDocument
	fpAuthorizer    refreshable.Authorizer
	providers       features.ProvidersClient
}

// Dynamic validates an OpenShift cluster
func (dv *openShiftClusterDynamicValidator) Dynamic(ctx context.Context) error {
	// Get all subnets
	mSubnetID := dv.oc.Properties.MasterProfile.SubnetID
	wSubnetIDs := []string{}

	for _, s := range dv.oc.Properties.WorkerProfiles {
		wSubnetIDs = append(wSubnetIDs, s.SubnetID)
	}

	// FP validation
	fpDynamic, err := NewValidator(dv.log, dv.env, mSubnetID, wSubnetIDs, dv.subscriptionDoc.ID, dv.fpAuthorizer, api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
	if err != nil {
		return err
	}

	err = fpDynamic.ValidateVnetPermissions(ctx)
	if err != nil {
		return err
	}

	err = fpDynamic.ValidateRouteTablesPermissions(ctx)
	if err != nil {
		return err
	}

	// SP validation
	spAuthorizer, err := validateServicePrincipalProfile(ctx, dv.log, dv.env, dv.oc, dv.subscriptionDoc)
	if err != nil {
		return err
	}

	spDynamic, err := NewValidator(dv.log, dv.env, mSubnetID, wSubnetIDs, dv.subscriptionDoc.ID, spAuthorizer, api.CloudErrorCodeInvalidServicePrincipalPermissions, "provided service principal")
	if err != nil {
		return err
	}

	err = spDynamic.ValidateVnetPermissions(ctx)
	if err != nil {
		return err
	}

	err = spDynamic.ValidateRouteTablesPermissions(ctx)
	if err != nil {
		return err
	}

	// Additional checks - use any dynamic because they both have the correct permissions
	err = spDynamic.ValidateVnetDNS(ctx)
	if err != nil {
		return err
	}

	vnet, err := spDynamic.virtualNetworks.Get(ctx, spDynamic.vnetr.ResourceGroup, spDynamic.vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	err = dv.validateCIDRRanges(ctx, &vnet)
	if err != nil {
		return err
	}

	err = dv.validateVnetLocation(ctx, &vnet)
	if err != nil {
		return err
	}

	err = dv.validateProviders(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (dv *openShiftClusterDynamicValidator) validateVnetLocation(ctx context.Context, vnet *mgmtnetwork.VirtualNetwork) error {
	dv.log.Print("validateVnetLocation")

	if !strings.EqualFold(*vnet.Location, dv.oc.Location) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet location '%s' must match the cluster location '%s'.", *vnet.Location, dv.oc.Location)
	}

	return nil
}

func (dv *openShiftClusterDynamicValidator) validateCIDRRanges(ctx context.Context, vnet *mgmtnetwork.VirtualNetwork) error {
	dv.log.Print("validateCIDRRanges")

	var CIDRArray []*net.IPNet

	for i, subnet := range dv.oc.Properties.WorkerProfiles {
		path := fmt.Sprintf("properties.workerProfiles[%d].subnetId", i)
		c, err := dv.validateSubnet(ctx, vnet, path, subnet.SubnetID)
		if err != nil {
			return err
		}

		CIDRArray = append(CIDRArray, c)
	}

	masterCIDR, err := dv.validateSubnet(ctx, vnet, "properties.MasterProfile.subnetId", dv.oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}
	_, podCIDR, err := net.ParseCIDR(dv.oc.Properties.NetworkProfile.PodCIDR)
	if err != nil {
		return err
	}

	_, serviceCIDR, err := net.ParseCIDR(dv.oc.Properties.NetworkProfile.ServiceCIDR)
	if err != nil {
		return err
	}

	CIDRArray = append(CIDRArray, masterCIDR, podCIDR, serviceCIDR)

	err = cidr.VerifyNoOverlap(CIDRArray, &net.IPNet{IP: net.IPv4zero, Mask: net.IPMask(net.IPv4zero)})
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided CIDRs must not overlap: '%s'.", err)
	}

	return nil
}

func (dv *openShiftClusterDynamicValidator) validateSubnet(ctx context.Context, vnet *mgmtnetwork.VirtualNetwork, path, subnetID string) (*net.IPNet, error) {
	dv.log.Printf("validateSubnet (%s)", path)

	s := findSubnet(vnet, subnetID)
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

func (dv *openShiftClusterDynamicValidator) validateProviders(ctx context.Context) error {
	dv.log.Print("validateProviders")

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

func validateServicePrincipalProfile(ctx context.Context, log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, sub *api.SubscriptionDocument) (refreshable.Authorizer, error) {
	log.Print("validateServicePrincipalProfile")

	token, err := aad.GetToken(ctx, log, oc, sub, env.Environment().ResourceManagerEndpoint)
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
