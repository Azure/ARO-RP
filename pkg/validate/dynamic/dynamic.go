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

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/Azure/checkaccess-v2-go-sdk/client"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armauthorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armmsi"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/token"
)

var (
	errMsgNSGAttached                        = "The provided subnet '%s' is invalid: must not have a network security group attached."
	errMsgOriginalNSGNotAttached             = "The provided subnet '%s' is invalid: must have network security group '%s' attached."
	errMsgNSGNotAttached                     = "The provided subnet '%s' is invalid: must have a network security group attached."
	errMsgNSGNotProperlyAttached             = "When the enable-preconfigured-nsg option is specified, both the master and worker subnets should have network security groups (NSG) attached to them before starting the cluster installation."
	errMsgSPHasNoRequiredPermissionsOnNSG    = "The %s service principal (Application ID: %s) does not have Network Contributor role on network security group '%s'. This is required when the enable-preconfigured-nsg option is specified."
	errMsgWIHasNoRequiredPermissionsOnNSG    = "The %s platform managed identity does not have required permissions on network security group '%s'. This is required when the enable-preconfigured-nsg option is specified."
	errMsgSubnetNotFound                     = "The provided subnet '%s' could not be found."
	errMsgSPHasNoRequiredPermissionsOnSubnet = "The %s service principal (Application ID: %s) does not have Network Contributor role on subnet '%s'."
	errMsgWIHasNoRequiredPermissionsOnSubnet = "The %s platform managed identity does not have required permissions on subnet '%s'."
	errMsgSubnetNotInSucceededState          = "The provided subnet '%s' is not in a Succeeded state"
	errMsgSubnetInvalidSize                  = "The provided subnet '%s' is invalid: must be /27 or larger."
	errMsgSPHasNoRequiredPermissionsOnVNet   = "The %s service principal (Application ID: %s) does not have Network Contributor role on vnet '%s'."
	errMsgWIHasNoRequiredPermissionsOnVNet   = "The %s platform managed identity does not have required permissions on vnet '%s'."
	errMsgVnetNotFound                       = "The vnet '%s' could not be found."
	errMsgSPHasNoRequiredPermissionsOnRT     = "The %s service principal does not have Network Contributor role on route table '%s'."
	errMsgWIHasNoRequiredPermissionsOnRT     = "The %s platform managed identity does not have required permissions on route table '%s'."
	errMsgRTNotFound                         = "The route table '%s' could not be found."
	errMsgSPHasNoRequiredPermissionsOnNatGW  = "The %s service principal does not have Network Contributor role on nat gateway '%s'."
	errMsgWIHasNoRequiredPermissionsOnNatGW  = "The %s platform managed identity does not have required permissions on nat gateway '%s'."
	errMsgNatGWNotFound                      = "The nat gateway '%s' could not be found."
	errMsgCIDROverlaps                       = "The provided CIDRs must not overlap: '%s'."
	errMsgInvalidVNetLocation                = "The vnet location '%s' must match the cluster location '%s'."
)

const minimumSubnetMaskSize int = 27

type Subnet struct {
	// ID is a resource id of the subnet
	ID string

	// Path is a path in the cluster document. For example, properties.workerProfiles[0].subnetId
	Path string
}

type ServicePrincipalValidator interface {
	ValidateServicePrincipal(ctx context.Context, spTokenCredential azcore.TokenCredential) error
}

// Dynamic validate in the operator context.
type Dynamic interface {
	ServicePrincipalValidator

	ValidateVnet(ctx context.Context, location string, subnets []Subnet, additionalCIDRs ...string) error
	ValidateSubnets(ctx context.Context, oc *api.OpenShiftCluster, subnets []Subnet) error
	ValidateDiskEncryptionSets(ctx context.Context, oc *api.OpenShiftCluster) error
	ValidateLoadBalancerProfile(ctx context.Context, oc *api.OpenShiftCluster) error
	ValidatePreConfiguredNSGs(ctx context.Context, oc *api.OpenShiftCluster, subnets []Subnet) error
	ValidateClusterUserAssignedIdentity(ctx context.Context, platformIdentities map[string]api.PlatformWorkloadIdentity, roleDefinitions armauthorization.RoleDefinitionsClient) error
	ValidatePlatformWorkloadIdentityProfile(
		ctx context.Context,
		oc *api.OpenShiftCluster,
		platformWorkloadIdentityRolesByRoleName map[string]api.PlatformWorkloadIdentityRole,
		roleDefinitions armauthorization.RoleDefinitionsClient,
		clusterMsiFederatedIdentityCredentials armmsi.FederatedIdentityCredentialsClient,
		platformWorkloadIdentities map[string]api.PlatformWorkloadIdentity,
	) error
}

type dynamic struct {
	log            *logrus.Entry
	appID          *string // for use when reporting an error
	authorizerType AuthorizerType
	// This represents the Subject for CheckAccess.  Could be either FP or SP.
	checkAccessSubjectInfoCred   azcore.TokenCredential
	env                          env.Interface
	azEnv                        *azureclient.AROEnvironment
	platformIdentities           map[string]api.PlatformWorkloadIdentity
	platformIdentitiesActionsMap map[string][]string

	virtualNetworks                       virtualNetworksGetClient
	diskEncryptionSets                    compute.DiskEncryptionSetsClient
	resourceSkusClient                    compute.ResourceSkusClient
	spNetworkUsage                        armnetwork.UsagesClient
	loadBalancerBackendAddressPoolsClient network.LoadBalancerBackendAddressPoolsClient
	pdpClient                             client.RemotePDPClient
}

type AuthorizerType string

const (
	AuthorizerFirstParty                  AuthorizerType = "resource provider"
	AuthorizerClusterServicePrincipal     AuthorizerType = "cluster"
	AuthorizerClusterUserAssignedIdentity AuthorizerType = "cluster user assigned identity"
	AuthorizerWorkloadIdentity            AuthorizerType = "platform workload identity"
)

func NewValidator(
	log *logrus.Entry,
	env env.Interface,
	azEnv *azureclient.AROEnvironment,
	subscriptionID string,
	authorizer autorest.Authorizer,
	appID *string,
	authorizerType AuthorizerType,
	cred azcore.TokenCredential,
	pdpClient client.RemotePDPClient,
) (Dynamic, error) {
	options := azEnv.ArmClientOptions()

	usagesClient, err := armnetwork.NewUsagesClient(subscriptionID, cred, options)
	if err != nil {
		return nil, err
	}

	virtualNetworksClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, options)
	if err != nil {
		return nil, err
	}

	return &dynamic{
		log:                        log,
		appID:                      appID,
		authorizerType:             authorizerType,
		env:                        env,
		azEnv:                      azEnv,
		checkAccessSubjectInfoCred: cred,

		spNetworkUsage:                        usagesClient,
		virtualNetworks:                       newVirtualNetworksCache(virtualNetworksClient),
		diskEncryptionSets:                    compute.NewDiskEncryptionSetsClientWithAROEnvironment(azEnv, subscriptionID, authorizer),
		resourceSkusClient:                    compute.NewResourceSkusClient(azEnv, subscriptionID, authorizer),
		pdpClient:                             pdpClient,
		loadBalancerBackendAddressPoolsClient: network.NewLoadBalancerBackendAddressPoolsClient(azEnv, subscriptionID, authorizer),
	}, nil
}

func NewServicePrincipalValidator(
	log *logrus.Entry,
	azEnv *azureclient.AROEnvironment,
	authorizerType AuthorizerType,
) ServicePrincipalValidator {
	return &dynamic{
		log:            log,
		authorizerType: authorizerType,
		azEnv:          azEnv,
	}
}

func (dv *dynamic) ValidateVnet(
	ctx context.Context,
	location string,
	subnets []Subnet,
	additionalCIDRs ...string,
) error {
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
		vnetID, _, err := apisubnet.Split(s.ID)
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
		err := dv.validateSubnetPermissions(ctx, s)
		if err != nil {
			return err
		}
	}

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

	operatorName, err := dv.validateActions(ctx, &vnet, []string{
		"Microsoft.Network/virtualNetworks/join/action",
		"Microsoft.Network/virtualNetworks/read",
		"Microsoft.Network/virtualNetworks/write",
	})

	var noPermissionsErr *api.CloudError
	if err != nil {
		if dv.authorizerType == AuthorizerWorkloadIdentity {
			noPermissionsErr = api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidWorkloadIdentityPermissions,
				"",
				fmt.Sprintf(
					errMsgWIHasNoRequiredPermissionsOnVNet,
					*operatorName,
					vnet.String(),
				))
		} else {
			noPermissionsErr = api.NewCloudError(
				http.StatusBadRequest,
				errCode,
				"",
				fmt.Sprintf(
					errMsgSPHasNoRequiredPermissionsOnVNet,
					dv.authorizerType,
					*dv.appID,
					vnet.String(),
				))
		}
	}

	if err == wait.ErrWaitTimeout {
		return noPermissionsErr
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		dv.log.Error(detailedErr)

		switch detailedErr.StatusCode {
		case http.StatusNotFound:
			return api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidLinkedVNet,
				"",
				fmt.Sprintf(
					errMsgVnetNotFound,
					vnet.String(),
				))
		case http.StatusForbidden:
			noPermissionsErr.Message = fmt.Sprintf(
				"%s\nOriginal error message: %s",
				noPermissionsErr.Message,
				detailedErr.Message,
			)
			return noPermissionsErr
		}
	}
	return err
}

func (dv *dynamic) validateSubnetPermissions(ctx context.Context, s Subnet) error {
	dv.log.Printf("validateSubnetPermissions")

	vnetID, _, err := apisubnet.Split(s.ID)
	if err != nil {
		return err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	subnetr, err := azure.ParseResourceID(s.ID)
	if err != nil {
		return err
	}

	// we need to explicitly set the resource type to virtualNetworks/{vnetID}/subnets, as the
	// ParseResourceID function does not properly handle parsing child resources (e.g.
	// VNET = parent, subnet = child) and gives the incorrect resource type, effectively
	// giving the incorrect resource ID which causes the validateActions method to fail.
	subnetr.ResourceType = fmt.Sprintf("%s/%s/%s", vnetr.ResourceType, vnetr.ResourceName, "subnets")

	errCode := api.CloudErrorCodeInvalidResourceProviderPermissions
	if dv.authorizerType == AuthorizerClusterServicePrincipal {
		errCode = api.CloudErrorCodeInvalidServicePrincipalPermissions
	}

	operatorName, err := dv.validateActions(ctx, &subnetr, []string{
		"Microsoft.Network/virtualNetworks/subnets/join/action",
		"Microsoft.Network/virtualNetworks/subnets/read",
		"Microsoft.Network/virtualNetworks/subnets/write",
	})

	var noPermissionsErr *api.CloudError
	if err != nil {
		if dv.authorizerType == AuthorizerWorkloadIdentity {
			noPermissionsErr = api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidWorkloadIdentityPermissions,
				"",
				fmt.Sprintf(
					errMsgWIHasNoRequiredPermissionsOnSubnet,
					*operatorName,
					subnetr.String(),
				))
		} else {
			noPermissionsErr = api.NewCloudError(
				http.StatusBadRequest,
				errCode,
				"",
				fmt.Sprintf(
					errMsgSPHasNoRequiredPermissionsOnSubnet,
					dv.authorizerType,
					*dv.appID,
					subnetr.String(),
				))
		}
	}

	if err == wait.ErrWaitTimeout {
		return noPermissionsErr
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		dv.log.Error(detailedErr)

		switch detailedErr.StatusCode {
		case http.StatusNotFound:
			return api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidLinkedSubnet,
				"",
				fmt.Sprintf(
					errMsgSubnetNotFound,
					subnetr.String(),
				))
		case http.StatusForbidden:
			noPermissionsErr.Message = fmt.Sprintf(
				"%s\nOriginal error message: %s",
				noPermissionsErr.Message,
				detailedErr.Message,
			)
			return noPermissionsErr
		}
	}
	return err
}

// validateRouteTablesPermissions will validate permissions on provided subnet
func (dv *dynamic) validateRouteTablePermissions(ctx context.Context, s Subnet) error {
	dv.log.Printf("validateRouteTablePermissions")

	vnetID, _, err := apisubnet.Split(s.ID)
	if err != nil {
		return err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	vnet, err := dv.virtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, nil)
	if err != nil {
		return err
	}

	rtID, err := getRouteTableID(&vnet.VirtualNetwork, s.ID)
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

	operatorName, err := dv.validateActions(ctx, &rtr, []string{
		"Microsoft.Network/routeTables/join/action",
		"Microsoft.Network/routeTables/read",
		"Microsoft.Network/routeTables/write",
	})
	if err == wait.ErrWaitTimeout {
		if dv.authorizerType == AuthorizerWorkloadIdentity {
			return api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidWorkloadIdentityPermissions,
				"",
				fmt.Sprintf(
					errMsgWIHasNoRequiredPermissionsOnRT,
					*operatorName,
					rtID,
				))
		}
		return api.NewCloudError(
			http.StatusBadRequest,
			errCode,
			"",
			fmt.Sprintf(
				errMsgSPHasNoRequiredPermissionsOnRT,
				dv.authorizerType,
				rtID,
			))
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidLinkedRouteTable,
			"",
			fmt.Sprintf(
				errMsgRTNotFound,
				rtID,
			))
	}
	return err
}

// validateNatGatewayPermissions will validate permissions on provided subnet
func (dv *dynamic) validateNatGatewayPermissions(ctx context.Context, s Subnet) error {
	dv.log.Printf("validateNatGatewayPermissions")

	vnetID, _, err := apisubnet.Split(s.ID)
	if err != nil {
		return err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	vnet, err := dv.virtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, nil)
	if err != nil {
		return err
	}

	ngID, err := getNatGatewayID(&vnet.VirtualNetwork, s.ID)
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

	operatorName, err := dv.validateActions(ctx, &ngr, []string{
		"Microsoft.Network/natGateways/join/action",
		"Microsoft.Network/natGateways/read",
		"Microsoft.Network/natGateways/write",
	})
	if err == wait.ErrWaitTimeout {
		if dv.authorizerType == AuthorizerWorkloadIdentity {
			return api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidWorkloadIdentityPermissions,
				"",
				fmt.Sprintf(
					errMsgWIHasNoRequiredPermissionsOnNatGW,
					*operatorName,
					ngID,
				))
		}
		return api.NewCloudError(
			http.StatusBadRequest,
			errCode,
			"",
			fmt.Sprintf(
				errMsgSPHasNoRequiredPermissionsOnNatGW,
				dv.authorizerType,
				ngID,
			))
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidLinkedNatGateway,
			"",
			fmt.Sprintf(
				errMsgNatGWNotFound,
				ngID,
			))
	}
	return err
}

// validateActionsByOID creates a closure with oid to call usingCheckAccessV2 for checking SP/MI has actions allowed on a resource
// oid is nil(fetched from access token) when validating FPSP, Non-MIWI Cluster Service Principal and MIWI Cluster User Assigned Managed Identity
// oid is passed only for validating MIWI Cluster Platform Managed Identity
func (dv *dynamic) validateActionsByOID(ctx context.Context, r *azure.Resource, actions []string, oid *string) error {
	// ARM has a 5 minute cache around role assignment creation, so wait one minute longer
	timeoutCtx, cancel := context.WithTimeout(ctx, 6*time.Minute)
	defer cancel()

	c := closure{dv: dv, ctx: ctx, resource: r, actions: actions, oid: oid}

	return wait.PollImmediateUntil(30*time.Second, c.usingCheckAccessV2, timeoutCtx.Done())
}

// closure is the closure used in PollImmediateUntil's ConditionalFunc
type closure struct {
	dv       *dynamic
	ctx      context.Context
	resource *azure.Resource
	actions  []string
	oid      *string
	jwtToken *string
}

func (c *closure) checkAccessAuthReqToken() error {
	scope := c.dv.env.Environment().ResourceManagerEndpoint + "/.default"
	t, err := c.dv.checkAccessSubjectInfoCred.GetToken(c.ctx, policy.TokenRequestOptions{Scopes: []string{scope}})
	if err != nil {
		c.dv.log.Error("Unable to get the token from AAD: ", err)
		return err
	}
	claims, err := token.ExtractClaims(t.Token)
	if err != nil {
		c.dv.log.Error("Unable to get the oid from token: ", err)
		return err
	}

	c.oid = &claims.ObjectId
	c.jwtToken = &t.Token
	return nil
}

// usingCheckAccessV2 uses the new RBAC checkAccessV2 API
func (c closure) usingCheckAccessV2() (result bool, err error) {
	c.dv.log.Info("validateActions with CheckAccessV2")

	var authReq *client.AuthorizationRequest
	//ensure token and oid is available during retries
	if c.dv.authorizerType != AuthorizerWorkloadIdentity {
		if c.jwtToken == nil || c.oid == nil {
			if err = c.checkAccessAuthReqToken(); err != nil {
				return false, err
			}
		}
		authReq, err = c.dv.pdpClient.CreateAuthorizationRequest(c.resource.String(), c.actions, *c.jwtToken)
		if err != nil {
			c.dv.log.Error("Unexpected error when creating CheckAccessV2 AuthorizationRequest: ", err)
			return false, err
		}
	} else {
		authReq = createAuthorizationRequestForPlatformWorkloadIdentity(*c.oid, c.resource.String(), c.actions...)
	}

	results, err := c.dv.pdpClient.CheckAccess(c.ctx, *authReq)
	if err != nil {
		c.dv.log.Error("Unexpected error when calling CheckAccessV2: ", err)
		return false, err
	}

	if results == nil {
		c.dv.log.Info("nil response returned from CheckAccessV2")
		return false, nil
	}

	actionsToFind := map[string]struct{}{}
	for _, action := range c.actions {
		actionsToFind[action] = struct{}{}
	}
	for _, result := range results.Value {
		_, ok := actionsToFind[result.ActionId]
		if ok {
			delete(actionsToFind, result.ActionId)
			if result.AccessDecision != client.Allowed {
				return false, nil
			}
		}
	}
	if len(actionsToFind) > 0 {
		c.dv.log.Infof("The result didn't include permissions %v for object ID: %s", actionsToFind, *c.oid)
		return false, nil
	}

	return true, nil
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
		vnetID, _, err := apisubnet.Split(s.ID)
		if err != nil {
			return err
		}

		vnetr, err := azure.ParseResourceID(vnetID)
		if err != nil {
			return err
		}

		vnet, err := dv.virtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, nil)
		if err != nil {
			return err
		}

		s, err := findSubnet(&vnet.VirtualNetwork, s.ID)
		if err != nil {
			return err
		}

		// Validate the CIDR of AddressPrefix or AddressPrefixes, whichever is defined
		if s.Properties.AddressPrefix == nil {
			for _, address := range s.Properties.AddressPrefixes {
				_, net, err := net.ParseCIDR(*address)
				if err != nil {
					return err
				}
				CIDRArray = append(CIDRArray, net)
			}
		} else {
			_, net, err := net.ParseCIDR(*s.Properties.AddressPrefix)
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
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidLinkedVNet,
			"",
			fmt.Sprintf(
				errMsgCIDROverlaps,
				err,
			))
	}

	return nil
}

func (dv *dynamic) validateVnetLocation(ctx context.Context, vnetr azure.Resource, location string) error {
	dv.log.Print("validateVnetLocation")

	vnet, err := dv.virtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, nil)
	if err != nil {
		return err
	}

	if !strings.EqualFold(*vnet.Location, location) {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidLinkedVNet,
			"",
			fmt.Sprintf(
				errMsgInvalidVNetLocation,
				*vnet.Location,
				location,
			))
	}

	return nil
}

func (dv *dynamic) createSubnetMapByID(ctx context.Context, subnets []Subnet) (map[string]*sdknetwork.Subnet, error) {
	if len(subnets) == 0 {
		return nil, fmt.Errorf("no subnets found")
	}
	subnetByID := make(map[string]*sdknetwork.Subnet)

	for _, s := range subnets {
		vnetID, _, err := apisubnet.Split(s.ID)
		if err != nil {
			return nil, err
		}
		vnetr, err := azure.ParseResourceID(vnetID)
		if err != nil {
			return nil, err
		}
		vnet, err := dv.virtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, nil)
		if err != nil {
			return nil, err
		}

		ss, err := findSubnet(&vnet.VirtualNetwork, s.ID)
		if err != nil {
			return nil, err
		}

		if ss == nil {
			return nil, api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidLinkedVNet,
				s.Path,
				fmt.Sprintf(
					errMsgSubnetNotFound,
					s.ID,
				))
		}

		subnetByID[s.ID] = ss
	}
	return subnetByID, nil
}

// checkPreconfiguredNSG checks whether all the subnets have an NSG attached.
// when the PreconfigureNSG feature flag is on and not all subnets are attached,
// it returns an error.
func (dv *dynamic) checkPreconfiguredNSG(subnetByID map[string]*sdknetwork.Subnet) error {
	var attached int
	for _, subnet := range subnetByID {
		if subnetHasNSGAttached(subnet) {
			attached++
		}
	}

	// all subnets have an attached NSG
	if attached == len(subnetByID) {
		dv.log.Info("all subnets are attached, BYO NSG")
		return nil // correct setup by customer
	}

	return &api.CloudError{
		StatusCode: http.StatusBadRequest,
		CloudErrorBody: &api.CloudErrorBody{
			Code:    api.CloudErrorCodeInvalidLinkedVNet,
			Message: errMsgNSGNotProperlyAttached,
		},
	}
}

func (dv *dynamic) ValidateSubnets(ctx context.Context, oc *api.OpenShiftCluster, subnets []Subnet) error {
	dv.log.Printf("validateSubnet")
	subnetByID, err := dv.createSubnetMapByID(ctx, subnets)
	if err != nil {
		return err
	}

	if oc.Properties.ProvisioningState == api.ProvisioningStateCreating {
		if oc.Properties.NetworkProfile.PreconfiguredNSG == api.PreconfiguredNSGEnabled {
			dv.log.Info("cluster creation with preconfigured-nsg")
			err = dv.checkPreconfiguredNSG(subnetByID)
			if err != nil {
				return err
			}
		}
	}

	// we're parsing through the subnets slice, not the map because we'll return consistent error messages on creation
	for _, s := range subnets {
		ss := subnetByID[s.ID]

		if oc.Properties.ProvisioningState == api.ProvisioningStateCreating {
			if subnetHasNSGAttached(ss) && oc.Properties.NetworkProfile.PreconfiguredNSG != api.PreconfiguredNSGEnabled {
				expectedNsgID, err := apisubnet.NetworkSecurityGroupID(oc, s.ID)
				if err != nil {
					return err
				}
				if !isTheSameNSG(*ss.Properties.NetworkSecurityGroup.ID, expectedNsgID) {
					return api.NewCloudError(
						http.StatusBadRequest,
						api.CloudErrorCodeInvalidLinkedVNet,
						s.Path, fmt.Sprintf(errMsgNSGAttached, s.ID))
				}
			}
		} else {
			nsgID, err := apisubnet.NetworkSecurityGroupID(oc, *ss.ID)
			if err != nil {
				return err
			}
			if oc.Properties.NetworkProfile.PreconfiguredNSG == api.PreconfiguredNSGDisabled {
				if !subnetHasNSGAttached(ss) ||
					!isTheSameNSG(*ss.Properties.NetworkSecurityGroup.ID, nsgID) {
					return api.NewCloudError(
						http.StatusBadRequest,
						api.CloudErrorCodeInvalidLinkedVNet,
						s.Path,
						fmt.Sprintf(
							errMsgOriginalNSGNotAttached,
							s.ID,
							nsgID,
						))
				}
			} else {
				if !subnetHasNSGAttached(ss) {
					return api.NewCloudError(
						http.StatusBadRequest,
						api.CloudErrorCodeInvalidLinkedVNet,
						s.Path,
						fmt.Sprintf(
							errMsgNSGNotAttached,
							s.ID,
						))
				}
			}
		}

		if ss.Properties == nil || ss.Properties.ProvisioningState == nil || *ss.Properties.ProvisioningState != sdknetwork.ProvisioningStateSucceeded {
			return api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidLinkedVNet,
				s.Path,
				fmt.Sprintf(
					errMsgSubnetNotInSucceededState,
					s.ID,
				))
		}

		// Handle both addressPrefix & addressPrefixes
		if ss.Properties.AddressPrefix == nil {
			for _, address := range ss.Properties.AddressPrefixes {
				if err = validateSubnetSize(s, *address); err != nil {
					return err
				}
			}
		} else {
			if err = validateSubnetSize(s, *ss.Properties.AddressPrefix); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateSubnetSize checks if the subnet mask is >27, and returns
// an error if so as it is too small for OCP
func validateSubnetSize(s Subnet, address string) error {
	_, net, err := net.ParseCIDR(address)
	if err != nil {
		return err
	}

	ones, _ := net.Mask.Size()
	if ones > minimumSubnetMaskSize {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidLinkedVNet,
			s.Path,
			fmt.Sprintf(
				errMsgSubnetInvalidSize,
				s.ID,
			))
	}
	return nil
}

func (dv *dynamic) ValidatePreConfiguredNSGs(ctx context.Context, oc *api.OpenShiftCluster, subnets []Subnet) error {
	dv.log.Print("ValidatePreConfiguredNSGs")

	if oc.Properties.NetworkProfile.PreconfiguredNSG != api.PreconfiguredNSGEnabled {
		return nil // exit early
	}

	subnetByID, err := dv.createSubnetMapByID(ctx, subnets)
	if err != nil {
		return err
	}

	for _, s := range subnetByID {
		nsgID := s.Properties.NetworkSecurityGroup.ID
		if nsgID == nil || *nsgID == "" {
			return api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeNotFound,
				"",
				errMsgNSGNotProperlyAttached,
			)
		}

		if err := dv.validateNSGPermissions(ctx, *nsgID); err != nil {
			return err
		}
	}
	return nil
}

// validateActions calls validateActionsByOID with object ID in case of MIWI cluster otherwise without object ID
func (dv *dynamic) validateActions(ctx context.Context, r *azure.Resource, actions []string) (*string, error) {
	if dv.platformIdentities != nil {
		for name, platformIdentity := range dv.platformIdentities {
			actionsToValidate := stringutils.GroupsIntersect(actions, dv.platformIdentitiesActionsMap[name])
			if len(actionsToValidate) > 0 {
				if err := dv.validateActionsByOID(ctx, r, actionsToValidate, &platformIdentity.ObjectID); err != nil {
					return &name, err
				}
			}
		}
	} else {
		if err := dv.validateActionsByOID(ctx, r, actions, nil); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (dv *dynamic) validateNSGPermissions(ctx context.Context, nsgID string) error {
	nsg, err := azure.ParseResourceID(nsgID)
	if err != nil {
		return err
	}

	operatorName, err := dv.validateActions(ctx, &nsg, []string{
		"Microsoft.Network/networkSecurityGroups/join/action",
	})

	if err == wait.ErrWaitTimeout {
		errCode := api.CloudErrorCodeInvalidResourceProviderPermissions
		switch dv.authorizerType {
		case AuthorizerClusterServicePrincipal:
			errCode = api.CloudErrorCodeInvalidServicePrincipalPermissions
		case AuthorizerWorkloadIdentity:
			return api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidWorkloadIdentityPermissions,
				"",
				fmt.Sprintf(
					errMsgWIHasNoRequiredPermissionsOnNSG,
					*operatorName,
					nsgID,
				))
		}
		return api.NewCloudError(
			http.StatusBadRequest,
			errCode,
			"",
			fmt.Sprintf(
				errMsgSPHasNoRequiredPermissionsOnNSG,
				dv.authorizerType,
				*dv.appID,
				nsgID,
			))
	}

	return err
}

func isTheSameNSG(found, inDB string) bool {
	return strings.EqualFold(found, inDB)
}

func subnetHasNSGAttached(subnet *sdknetwork.Subnet) bool {
	return subnet.Properties.NetworkSecurityGroup != nil && subnet.Properties.NetworkSecurityGroup.ID != nil
}

func getRouteTableID(vnet *sdknetwork.VirtualNetwork, subnetID string) (string, error) {
	s, err := findSubnet(vnet, subnetID)
	if err != nil {
		return "", err
	}

	if s == nil || s.Properties.RouteTable == nil {
		return "", nil
	}

	return *s.Properties.RouteTable.ID, nil
}

func getNatGatewayID(vnet *sdknetwork.VirtualNetwork, subnetID string) (string, error) {
	s, err := findSubnet(vnet, subnetID)
	if err != nil {
		return "", err
	}

	if s == nil || s.Properties.NatGateway == nil {
		return "", nil
	}

	return *s.Properties.NatGateway.ID, nil
}

func findSubnet(vnet *sdknetwork.VirtualNetwork, subnetID string) (*sdknetwork.Subnet, error) {
	if vnet.Properties.Subnets != nil {
		for _, s := range vnet.Properties.Subnets {
			if strings.EqualFold(*s.ID, subnetID) {
				return s, nil
			}
		}
	}

	return nil, api.NewCloudError(
		http.StatusBadRequest,
		api.CloudErrorCodeInvalidLinkedVNet,
		"",
		fmt.Sprintf(
			errMsgSubnetNotFound,
			subnetID,
		))
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

func createAuthorizationRequestForPlatformWorkloadIdentity(subject, resourceId string, actions ...string) *client.AuthorizationRequest {
	actionInfos := []client.ActionInfo{}
	for _, action := range actions {
		actionInfos = append(actionInfos, client.ActionInfo{Id: action})
	}

	return &client.AuthorizationRequest{
		Subject: client.SubjectInfo{
			Attributes: client.SubjectAttributes{
				ObjectId: subject,
			},
		},
		Actions: actionInfos,
		Resource: client.ResourceInfo{
			Id: resourceId,
		},
	}
}
