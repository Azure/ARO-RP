package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate/dynamic/vnetcache"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	networkutil "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type VnetValidator interface {
	Validate(ctx context.Context, location string, subnets []Subnet, oc *api.OpenShiftCluster) error
}

type defaultVnetValidator struct {
	log             *logrus.Entry
	permissionsSP   authorization.PermissionsClient
	permissionsFP   authorization.PermissionsClient
	networkSP       networkutil.VirtualNetworksClient
	networkFP       networkutil.VirtualNetworksClient
	virtualNetworks vnetcache.VirtualNetworksGetClient
}

func NewVnetValidator(log *logrus.Entry, permsFP, permsSP authorization.PermissionsClient, networkFP, networkSP networkutil.VirtualNetworksClient) *defaultVnetValidator {
	return &defaultVnetValidator{
		log:           log,
		permissionsFP: permsFP,
		permissionsSP: permsSP,
		networkSP:     networkSP,
		networkFP:     networkFP,
	}
}

func (dv *defaultVnetValidator) Validate(ctx context.Context, location string, subnets []Subnet, oc *api.OpenShiftCluster) error {
	dv.virtualNetworks = vnetcache.NewVirtualNetworksCache(dv.networkFP)
	cidrs := []string{oc.Properties.NetworkProfile.PodCIDR, oc.Properties.NetworkProfile.ServiceCIDR}
	err := dv.validateOne(ctx, location, subnets, dv.permissionsFP, AuthorizerFirstParty, cidrs...)
	if err != nil {
		return err
	}
	dv.virtualNetworks = vnetcache.NewVirtualNetworksCache(dv.networkSP)
	return dv.validateOne(ctx, location, subnets, dv.permissionsSP, AuthorizerClusterServicePrincipal, cidrs...)
}

func (dv *defaultVnetValidator) validateOne(ctx context.Context, location string, subnets []Subnet, permsClient authorization.PermissionsClient, authType AuthorizerType, additionalCIDRs ...string) error {
	if len(subnets) == 0 {
		return fmt.Errorf("no subnets provided")
	}

	// each subnet is threated individually as it would be from the different vnet
	// During cluster runtime worker profile gets enriched and contains multiple
	// duplicate values for multiple worker pools. We care only about
	// unique subnet value in the functions below.
	subnets = uniqueSubnetSlice(subnets)

	// get unique vnets from subnets
	vnets := make(map[string]azure.Resource)
	for _, s := range subnets {
		vnetID, _, err := subnet.Split(s.ID)
		if err != nil {
			return err
		}

		vnetr, err := azure.ParseResourceID(vnetID)
		if err != nil {
			return err
		}
		vnets[strings.ToLower(vnetID)] = vnetr
	}

	// validate at vnet level
	for _, vnet := range vnets {
		err := dv.validateVnetPermissions(ctx, vnet, permsClient, authType)
		if err != nil {
			return err
		}

		err = dv.validateVnetLocation(ctx, vnet, location)
		if err != nil {
			return err
		}
	}

	// validate at subnets level
	for _, s := range subnets {
		err := dv.validateRouteTablePermissions(ctx, s, permsClient, authType)
		if err != nil {
			return err
		}
	}

	for _, s := range subnets {
		err := dv.validateNatGatewayPermissions(ctx, s, permsClient, authType)
		if err != nil {
			return err
		}
	}
	return dv.validateCIDRRanges(ctx, subnets, additionalCIDRs...)
}

func (dv *defaultVnetValidator) validateVnetPermissions(ctx context.Context, vnet azure.Resource, permsClient authorization.PermissionsClient, authType AuthorizerType) error {
	dv.log.Printf("validateVnetPermissions")

	errCode := api.CloudErrorCodeInvalidResourceProviderPermissions
	if authType == AuthorizerClusterServicePrincipal {
		errCode = api.CloudErrorCodeInvalidServicePrincipalPermissions
	}

	err := validateActions(
		ctx,
		dv.log,
		permsClient,
		&vnet,
		[]string{
			"Microsoft.Network/virtualNetworks/join/action",
			"Microsoft.Network/virtualNetworks/read",
			"Microsoft.Network/virtualNetworks/write",
			"Microsoft.Network/virtualNetworks/subnets/join/action",
			"Microsoft.Network/virtualNetworks/subnets/read",
			"Microsoft.Network/virtualNetworks/subnets/write",
		},
	)

	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, errCode, "", "The %s service principal does not have Network Contributor permission on vnet '%s'.", authType, vnet.String())
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet '%s' could not be found.", vnet.String())
	}
	return err
}

// validateRouteTablesPermissions will validate permissions on provided subnet
func (dv *defaultVnetValidator) validateRouteTablePermissions(ctx context.Context, s Subnet, permsClient authorization.PermissionsClient, authType AuthorizerType) error {
	dv.log.Printf("validateRouteTablePermissions")

	vnetID, _, err := subnet.Split(s.ID)
	if err != nil {
		return err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	vnet, err := dv.virtualNetworks.Get(ctx, vnetcache.CacheKeyFromResource(vnetr))
	if err != nil {
		return err
	}

	rtID, err := getRouteTableID(&vnet, s.ID)
	if err != nil || rtID == "" { // error or no route table
		return err
	}

	rtr, err := azure.ParseResourceID(rtID)
	if err != nil {
		return err
	}

	errCode := api.CloudErrorCodeInvalidResourceProviderPermissions
	if authType == AuthorizerClusterServicePrincipal {
		errCode = api.CloudErrorCodeInvalidServicePrincipalPermissions
	}

	err = validateActions(
		ctx,
		dv.log,
		permsClient,
		&rtr,
		[]string{"Microsoft.Network/routeTables/join/action", "Microsoft.Network/routeTables/read", "Microsoft.Network/routeTables/write"},
	)

	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, errCode, "", "The %s service principal does not have Network Contributor permission on route table '%s'.", authType, rtID)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedRouteTable, "", "The route table '%s' could not be found.", rtID)
	}
	return err
}

// validateNatGatewayPermissions will validate permissions on provided subnet
func (dv *defaultVnetValidator) validateNatGatewayPermissions(ctx context.Context, s Subnet, permsClient authorization.PermissionsClient, authType AuthorizerType) error {
	dv.log.Printf("validateNatGatewayPermissions")

	vnetID, _, err := subnet.Split(s.ID)
	if err != nil {
		return err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	vnet, err := dv.virtualNetworks.Get(ctx, vnetcache.CacheKeyFromResource(vnetr))
	if err != nil {
		return err
	}

	ngID, err := getNatGatewayID(&vnet, s.ID)
	if err != nil {
		return err
	}

	if ngID == "" { // empty nat gateway
		return nil
	}

	ngr, err := azure.ParseResourceID(ngID)
	if err != nil {
		return err
	}

	errCode := api.CloudErrorCodeInvalidResourceProviderPermissions
	if authType == AuthorizerClusterServicePrincipal {
		errCode = api.CloudErrorCodeInvalidServicePrincipalPermissions
	}

	err = validateActions(
		ctx,
		dv.log,
		permsClient,
		&ngr,
		[]string{"Microsoft.Network/natGateways/join/action", "Microsoft.Network/natGateways/read", "Microsoft.Network/natGateways/write"},
	)
	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, errCode, "", "The %s service principal does not have Network Contributor permission on nat gateway '%s'.", authType, ngID)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedNatGateway, "", "The nat gateway '%s' could not be found.", ngID)
	}
	return err
}

func (dv *defaultVnetValidator) validateCIDRRanges(ctx context.Context, subnets []Subnet, additionalCIDRs ...string) error {
	dv.log.Print("ValidateCIDRRanges")

	// During cluster runtime they get enriched and contains multiple
	// duplicate values for multiple worker pools. CIDRRange validation
	// only cares about unique CIDR ranges.
	subnets = uniqueSubnetSlice(subnets)

	var CIDRArray []*net.IPNet

	// unique names of subnets from all node pools
	for _, s := range subnets {
		vnetID, _, err := subnet.Split(s.ID)
		if err != nil {
			return err
		}

		vnetr, err := azure.ParseResourceID(vnetID)
		if err != nil {
			return err
		}

		vnet, err := dv.virtualNetworks.Get(ctx, vnetcache.CacheKeyFromResource(vnetr))
		if err != nil {
			return err
		}

		s, err := findSubnet(&vnet, s.ID)
		if err != nil {
			return err
		}

		// Validate the CIDR of AddressPrefix or AddressPrefixes, whichever is defined
		if s.AddressPrefix == nil {
			for _, address := range *s.AddressPrefixes {
				_, net, err := net.ParseCIDR(address)
				if err != nil {
					return err
				}
				CIDRArray = append(CIDRArray, net)
			}
		} else {
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

	err := cidr.VerifyNoOverlap(CIDRArray, &net.IPNet{IP: net.IPv4zero, Mask: net.IPMask(net.IPv4zero)})
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided CIDRs must not overlap: '%s'.", err)
	}

	return nil
}

func (dv *defaultVnetValidator) validateVnetLocation(ctx context.Context, vnetr azure.Resource, location string) error {
	dv.log.Print("validateVnetLocation")

	vnet, err := dv.virtualNetworks.Get(ctx, vnetcache.CacheKeyFromResource(vnetr))
	if err != nil {
		return err
	}

	if !strings.EqualFold(*vnet.Location, location) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet location '%s' must match the cluster location '%s'.", *vnet.Location, location)
	}

	return nil
}
