package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/authz/remotepdp"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/permissions"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type Subnet struct {
	// ID is a resource id of the subnet
	ID string

	// Path is a path in the cluster document. For example, properties.workerProfiles[0].subnetId
	Path string
}

type ServicePrincipalValidator interface {
	ValidateServicePrincipal(ctx context.Context, clientID, clientSecret, tenantID string) error
}

// Dynamic validate in the operator context.
type Dynamic interface {
	ServicePrincipalValidator

	ValidateVnet(ctx context.Context, location string, subnets []Subnet, additionalCIDRs ...string) error
	ValidateSubnets(ctx context.Context, oc *api.OpenShiftCluster, subnets []Subnet) error
	ValidateProviders(ctx context.Context) error
	ValidateDiskEncryptionSets(ctx context.Context, oc *api.OpenShiftCluster) error
	ValidateEncryptionAtHost(ctx context.Context, oc *api.OpenShiftCluster) error
}

type dynamic struct {
	log            *logrus.Entry
	authorizer     refreshable.Authorizer
	authorizerType AuthorizerType
	env            env.Interface
	azEnv          *azureclient.AROEnvironment

	permissions        authorization.PermissionsClient
	providers          features.ProvidersClient
	virtualNetworks    virtualNetworksGetClient
	diskEncryptionSets compute.DiskEncryptionSetsClient
	resourceSkusClient compute.ResourceSkusClient
	spComputeUsage     compute.UsageClient
	spNetworkUsage     network.UsageClient
	tokenClient        aad.TokenClient
	pdpClient          remotepdp.RemotePDPClient
}

type AuthorizerType string

const (
	AuthorizerFirstParty              AuthorizerType = "resource provider"
	AuthorizerClusterServicePrincipal AuthorizerType = "cluster"
)

func NewValidator(log *logrus.Entry, env env.Interface, azEnv *azureclient.AROEnvironment, subscriptionID string, authorizer refreshable.Authorizer, authorizerType AuthorizerType, tokenClient aad.TokenClient, pdpClient remotepdp.RemotePDPClient) (Dynamic, error) {
	return &dynamic{
		log:            log,
		authorizer:     authorizer,
		authorizerType: authorizerType,
		env:            env,
		azEnv:          azEnv,

		providers:          features.NewProvidersClient(azEnv, subscriptionID, authorizer),
		spComputeUsage:     compute.NewUsageClient(azEnv, subscriptionID, authorizer),
		spNetworkUsage:     network.NewUsageClient(azEnv, subscriptionID, authorizer),
		permissions:        authorization.NewPermissionsClient(azEnv, subscriptionID, authorizer),
		virtualNetworks:    newVirtualNetworksCache(network.NewVirtualNetworksClient(azEnv, subscriptionID, authorizer)),
		diskEncryptionSets: compute.NewDiskEncryptionSetsClient(azEnv, subscriptionID, authorizer),
		resourceSkusClient: compute.NewResourceSkusClient(azEnv, subscriptionID, authorizer),
		tokenClient:        tokenClient,
		pdpClient:          pdpClient,
	}, nil
}

func NewServicePrincipalValidator(log *logrus.Entry, azEnv *azureclient.AROEnvironment, authorizerType AuthorizerType, tokenClient aad.TokenClient) (ServicePrincipalValidator, error) {
	return &dynamic{
		log:            log,
		authorizerType: authorizerType,
		azEnv:          azEnv,
		tokenClient:    tokenClient,
	}, nil
}

func (dv *dynamic) ValidateVnet(ctx context.Context, location string, subnets []Subnet, additionalCIDRs ...string) error {
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
		err := dv.validateVnetPermissions(ctx, vnet)
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
		err := dv.validateRouteTablePermissions(ctx, s)
		if err != nil {
			return err
		}
	}

	for _, s := range subnets {
		err := dv.validateNatGatewayPermissions(ctx, s)
		if err != nil {
			return err
		}
	}
	return dv.validateCIDRRanges(ctx, subnets, additionalCIDRs...)
}

func (dv *dynamic) validateVnetPermissions(ctx context.Context, vnet azure.Resource) error {
	dv.log.Printf("validateVnetPermissions")

	errCode := api.CloudErrorCodeInvalidResourceProviderPermissions
	if dv.authorizerType == AuthorizerClusterServicePrincipal {
		errCode = api.CloudErrorCodeInvalidServicePrincipalPermissions
	}

	err := dv.validateActions(ctx, &vnet, []string{
		"Microsoft.Network/virtualNetworks/join/action",
		"Microsoft.Network/virtualNetworks/read",
		"Microsoft.Network/virtualNetworks/write",
		"Microsoft.Network/virtualNetworks/subnets/join/action",
		"Microsoft.Network/virtualNetworks/subnets/read",
		"Microsoft.Network/virtualNetworks/subnets/write",
	})

	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, errCode, "", "The %s service principal does not have Network Contributor permission on vnet '%s'.", dv.authorizerType, vnet.String())
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet '%s' could not be found.", vnet.String())
	}
	return err
}

// validateRouteTablesPermissions will validate permissions on provided subnet
func (dv *dynamic) validateRouteTablePermissions(ctx context.Context, s Subnet) error {
	dv.log.Printf("validateRouteTablePermissions")

	vnetID, _, err := subnet.Split(s.ID)
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

	rtID, err := getRouteTableID(&vnet, s.ID)
	if err != nil || rtID == "" { // error or no route table
		return err
	}

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

// validateNatGatewayPermissions will validate permissions on provided subnet
func (dv *dynamic) validateNatGatewayPermissions(ctx context.Context, s Subnet) error {
	dv.log.Printf("validateNatGatewayPermissions")

	vnetID, _, err := subnet.Split(s.ID)
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
	if dv.authorizerType == AuthorizerClusterServicePrincipal {
		errCode = api.CloudErrorCodeInvalidServicePrincipalPermissions
	}

	err = dv.validateActions(ctx, &ngr, []string{
		"Microsoft.Network/natGateways/join/action",
		"Microsoft.Network/natGateways/read",
		"Microsoft.Network/natGateways/write",
	})
	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, errCode, "", "The %s service principal does not have Network Contributor permission on nat gateway '%s'.", dv.authorizerType, ngID)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedNatGateway, "", "The nat gateway '%s' could not be found.", ngID)
	}
	return err
}

func (dv *dynamic) validateActions(ctx context.Context, r *azure.Resource, actions []string) error {
	c := closure{dv: dv, ctx: ctx, resource: r, actions: actions}
	conditionalFunc := c.usingListPermissions
	timeout := 5 * time.Minute
	if dv.pdpClient != nil {
		conditionalFunc = c.usingCheckAccessV2
		timeout = 185 * time.Second // checkAccess refreshes data every min. This allows ~3 retries.
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return wait.PollImmediateUntil(timeout, conditionalFunc, timeoutCtx.Done())
}

// closure is the closure used in PollImmediateUntil's ConditionalFunc
type closure struct {
	dv       *dynamic
	ctx      context.Context
	resource *azure.Resource
	actions  []string
}

// usingListPermissions is how the current check is done
func (c closure) usingListPermissions() (bool, error) {
	c.dv.log.Debug("retry validateActions with ListPermissions")
	perms, err := c.dv.permissions.ListForResource(c.ctx, c.resource.ResourceGroup, c.resource.Provider, "", c.resource.ResourceType, c.resource.ResourceName)

	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusForbidden {
		return false, steps.ErrWantRefresh
	}
	if err != nil {
		return false, err
	}

	for _, action := range c.actions {
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
}

// usingCheckAccessV2 uses the new RBAC checkAccessV2 API
func (c closure) usingCheckAccessV2() (bool, error) {
	c.dv.log.Info("retry validationActions with CheckAccessV2")
	oid, err := aad.GetObjectId(c.dv.authorizer.OAuthToken())
	if err != nil {
		return false, err
	}

	authReq := createAuthorizationRequest(oid, c.resource.String(), c.actions...)
	results, err := c.dv.pdpClient.CheckAccess(c.ctx, authReq)
	if err != nil {
		return false, err
	}

	for _, result := range results.Value {
		if result.AccessDecision != remotepdp.Allowed {
			c.dv.log.Debug(fmt.Sprintf("%s has no access to %s", oid, result.ActionId))
			return false, nil
		}
	}

	return true, nil
}

func createAuthorizationRequest(subject, resourceId string, actions ...string) remotepdp.AuthorizationRequest {
	actionInfos := []remotepdp.ActionInfo{}
	for _, action := range actions {
		actionInfos = append(actionInfos, remotepdp.ActionInfo{Id: action})
	}

	return remotepdp.AuthorizationRequest{
		Subject: remotepdp.SubjectInfo{
			Attributes: remotepdp.SubjectAttributes{
				ObjectId: subject,
			},
		},
		Actions: actionInfos,
		Resource: remotepdp.ResourceInfo{
			Id: resourceId,
		},
	}
}

func (dv *dynamic) validateCIDRRanges(ctx context.Context, subnets []Subnet, additionalCIDRs ...string) error {
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

		vnet, err := dv.virtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, "")
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

func (dv *dynamic) validateVnetLocation(ctx context.Context, vnetr azure.Resource, location string) error {
	dv.log.Print("validateVnetLocation")

	vnet, err := dv.virtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	if !strings.EqualFold(*vnet.Location, location) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet location '%s' must match the cluster location '%s'.", *vnet.Location, location)
	}

	return nil
}

func (dv *dynamic) ValidateSubnets(ctx context.Context, oc *api.OpenShiftCluster, subnets []Subnet) error {
	dv.log.Printf("validateSubnet")
	if len(subnets) == 0 {
		return fmt.Errorf("no subnets found")
	}

	for _, s := range subnets {
		vnetID, _, err := subnet.Split(s.ID)
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

		ss, err := findSubnet(&vnet, s.ID)
		if err != nil {
			return err
		}
		if ss == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, s.Path, "The provided subnet '%s' could not be found.", s.ID)
		}

		if oc.Properties.ProvisioningState == api.ProvisioningStateCreating {
			if ss.SubnetPropertiesFormat != nil &&
				ss.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
				expectedNsgID, err := subnet.NetworkSecurityGroupID(oc, s.ID)
				if err != nil {
					return err
				}

				if !strings.EqualFold(*ss.SubnetPropertiesFormat.NetworkSecurityGroup.ID, expectedNsgID) {
					return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, s.Path, "The provided subnet '%s' is invalid: must not have a network security group attached.", s.ID)
				}
			}
		} else {
			nsgID, err := subnet.NetworkSecurityGroupID(oc, *ss.ID)
			if err != nil {
				return err
			}

			if ss.SubnetPropertiesFormat == nil ||
				ss.SubnetPropertiesFormat.NetworkSecurityGroup == nil ||
				!strings.EqualFold(*ss.SubnetPropertiesFormat.NetworkSecurityGroup.ID, nsgID) {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, s.Path, "The provided subnet '%s' is invalid: must have network security group '%s' attached.", s.ID, nsgID)
			}
		}

		if ss.SubnetPropertiesFormat == nil ||
			ss.SubnetPropertiesFormat.ProvisioningState != mgmtnetwork.Succeeded {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, s.Path, "The provided subnet '%s' is not in a Succeeded state", s.ID)
		}

		_, net, err := net.ParseCIDR(*ss.AddressPrefix)
		if err != nil {
			return err
		}

		ones, _ := net.Mask.Size()
		if ones > 27 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, s.Path, "The provided subnet '%s' is invalid: must be /27 or larger.", s.ID)
		}
	}

	return nil
}

func getRouteTableID(vnet *mgmtnetwork.VirtualNetwork, subnetID string) (string, error) {
	s, err := findSubnet(vnet, subnetID)
	if err != nil {
		return "", err
	}

	if s == nil || s.RouteTable == nil {
		return "", nil
	}

	return *s.RouteTable.ID, nil
}

func getNatGatewayID(vnet *mgmtnetwork.VirtualNetwork, subnetID string) (string, error) {
	s, err := findSubnet(vnet, subnetID)
	if err != nil {
		return "", err
	}

	if s == nil || s.NatGateway == nil {
		return "", nil
	}

	return *s.NatGateway.ID, nil
}

func findSubnet(vnet *mgmtnetwork.VirtualNetwork, subnetID string) (*mgmtnetwork.Subnet, error) {
	if vnet.Subnets != nil {
		for _, s := range *vnet.Subnets {
			if strings.EqualFold(*s.ID, subnetID) {
				return &s, nil
			}
		}
	}

	return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided subnet '%s' could not be found.", subnetID)
}

// uniqueSubnetSlice returns string subnets with unique values only
func uniqueSubnetSlice(slice []Subnet) []Subnet {
	keys := make(map[string]bool)
	list := []Subnet{}
	for _, entry := range slice {
		if _, value := keys[strings.ToLower(entry.ID)]; !value {
			keys[strings.ToLower(entry.ID)] = true
			list = append(list, entry)
		}
	}
	return list
}
