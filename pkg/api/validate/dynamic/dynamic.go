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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/authz/remotepdp"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/permissions"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/token"
)

type Subnet struct {
	// ID is a resource id of the subnet
	ID string

	// Path is a path in the cluster document. For example, properties.workerProfiles[0].subnetId
	Path string
}

type ServicePrincipalValidator interface {
	ValidateServicePrincipal(ctx context.Context, tokenCredential azcore.TokenCredential) error
}

// Dynamic validate in the operator context.
type Dynamic interface {
	ServicePrincipalValidator

	ValidateVnet(ctx context.Context, location string, subnets []Subnet, additionalCIDRs ...string) error
	ValidateSubnets(ctx context.Context, oc *api.OpenShiftCluster, subnets []Subnet) error
	ValidateDiskEncryptionSets(ctx context.Context, oc *api.OpenShiftCluster) error
	ValidateEncryptionAtHost(ctx context.Context, oc *api.OpenShiftCluster) error
}

type dynamic struct {
	log            *logrus.Entry
	appID          string // for use when reporting an error
	authorizerType AuthorizerType
	// This represents the Subject for CheckAccess.  Could be either FP or SP.
	checkAccessSubjectInfoCred azcore.TokenCredential
	env                        env.Interface
	azEnv                      *azureclient.AROEnvironment

	permissions        authorization.PermissionsClient
	virtualNetworks    virtualNetworksGetClient
	diskEncryptionSets compute.DiskEncryptionSetsClient
	resourceSkusClient compute.ResourceSkusClient
	spComputeUsage     compute.UsageClient
	spNetworkUsage     network.UsageClient
	pdpClient          remotepdp.RemotePDPClient
}

type AuthorizerType string

const (
	AuthorizerFirstParty              AuthorizerType = "resource provider"
	AuthorizerClusterServicePrincipal AuthorizerType = "cluster"
)

func NewValidator(
	log *logrus.Entry,
	env env.Interface,
	azEnv *azureclient.AROEnvironment,
	subscriptionID string,
	authorizer autorest.Authorizer,
	appID string,
	authorizerType AuthorizerType,
	cred azcore.TokenCredential,
	pdpClient remotepdp.RemotePDPClient,
) Dynamic {
	return &dynamic{
		log:                        log,
		appID:                      appID,
		authorizerType:             authorizerType,
		env:                        env,
		azEnv:                      azEnv,
		checkAccessSubjectInfoCred: cred,

		spComputeUsage: compute.NewUsageClient(azEnv, subscriptionID, authorizer),
		spNetworkUsage: network.NewUsageClient(azEnv, subscriptionID, authorizer),
		permissions:    authorization.NewPermissionsClient(azEnv, subscriptionID, authorizer),
		virtualNetworks: newVirtualNetworksCache(
			network.NewVirtualNetworksClient(azEnv, subscriptionID, authorizer),
		),
		diskEncryptionSets: compute.NewDiskEncryptionSetsClient(azEnv, subscriptionID, authorizer),
		resourceSkusClient: compute.NewResourceSkusClient(azEnv, subscriptionID, authorizer),
		pdpClient:          pdpClient,
	}
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

	noPermissionsErr := api.NewCloudError(
		http.StatusBadRequest,
		errCode,
		"",
		"The %s service principal (Application ID: %s) does not have Network Contributor role on vnet '%s'.",
		dv.authorizerType,
		dv.appID,
		vnet.String(),
	)

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
				"The vnet '%s' could not be found.",
				vnet.String(),
			)
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
		return api.NewCloudError(
			http.StatusBadRequest,
			errCode,
			"",
			"The %s service principal does not have Network Contributor role on route table '%s'.",
			dv.authorizerType,
			rtID,
		)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidLinkedRouteTable,
			"",
			"The route table '%s' could not be found.",
			rtID,
		)
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
		return api.NewCloudError(
			http.StatusBadRequest,
			errCode,
			"",
			"The %s service principal does not have Network Contributor role on nat gateway '%s'.",
			dv.authorizerType,
			ngID,
		)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidLinkedNatGateway,
			"",
			"The nat gateway '%s' could not be found.",
			ngID,
		)
	}
	return err
}

func (dv *dynamic) validateActions(ctx context.Context, r *azure.Resource, actions []string) error {
	// ARM has a 5 minute cache around role assignment creation, so wait one minute longer
	timeoutCtx, cancel := context.WithTimeout(ctx, 6*time.Minute)
	defer cancel()

	c := closure{dv: dv, ctx: ctx, resource: r, actions: actions}
	conditionalFunc := c.usingListPermissions
	if dv.pdpClient != nil {
		conditionalFunc = c.usingCheckAccessV2
	}

	return wait.PollImmediateUntil(30*time.Second, conditionalFunc, timeoutCtx.Done())
}

// closure is the closure used in PollImmediateUntil's ConditionalFunc
type closure struct {
	dv       *dynamic
	ctx      context.Context
	resource *azure.Resource
	actions  []string
	oid      *string
}

// usingListPermissions is how the current check is done
func (c closure) usingListPermissions() (bool, error) {
	c.dv.log.Debug("retry validateActions with ListPermissions")
	perms, err := c.dv.permissions.ListForResource(
		c.ctx,
		c.resource.ResourceGroup,
		c.resource.Provider,
		"",
		c.resource.ResourceType,
		c.resource.ResourceName,
	)
	if err != nil {
		return false, err
	}

	// If we get a StatusForbidden, try refreshing the SP (since with a
	// brand-new SP it might take time to propagate)
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
	// TODO remove this when fully migrated to CheckAccess
	c.dv.log.Info("validateActions with CheckAccessV2")

	// reusing oid during retries
	if c.oid == nil {
		scope := c.dv.azEnv.ResourceManagerEndpoint + "/.default"
		t, err := c.dv.checkAccessSubjectInfoCred.GetToken(c.ctx, policy.TokenRequestOptions{Scopes: []string{scope}})
		if err != nil {
			c.dv.log.Error("Unable to get the token from AAD", err)
			return false, err
		}

		oid, err := token.GetObjectId(t.Token)
		if err != nil {
			c.dv.log.Error("Unable to parse the token oid claim")
			return false, err
		}
		c.oid = &oid
	}

	authReq := createAuthorizationRequest(*c.oid, c.resource.String(), c.actions...)
	results, err := c.dv.pdpClient.CheckAccess(c.ctx, authReq)
	if err != nil {
		c.dv.log.Error("Unexpected error when calling CheckAccessV2", err)
		return false, err
	}

	if results == nil {
		c.dv.log.Info("nil response returned from CheckAccessV2")
		return false, nil
	}

	for _, action := range c.actions {
		found := false
		for _, result := range results.Value {
			if result.ActionId == action {
				found = true
				if result.AccessDecision == remotepdp.Allowed {
					break
				}
				return false, nil
			}
		}
		if !found {
			c.dv.log.Infof("The result didn't include permission %s", action)
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
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidLinkedVNet,
			"",
			"The provided CIDRs must not overlap: '%s'.",
			err,
		)
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
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidLinkedVNet,
			"",
			"The vnet location '%s' must match the cluster location '%s'.",
			*vnet.Location,
			location,
		)
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
			return api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidLinkedVNet,
				s.Path,
				"The provided subnet '%s' could not be found.",
				s.ID,
			)
		}

		if oc.Properties.ProvisioningState == api.ProvisioningStateCreating {
			if ss.SubnetPropertiesFormat != nil &&
				ss.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
				expectedNsgID, err := subnet.NetworkSecurityGroupID(oc, s.ID)
				if err != nil {
					return err
				}

				if !strings.EqualFold(*ss.SubnetPropertiesFormat.NetworkSecurityGroup.ID, expectedNsgID) {
					return api.NewCloudError(
						http.StatusBadRequest,
						api.CloudErrorCodeInvalidLinkedVNet,
						s.Path,
						"The provided subnet '%s' is invalid: must not have a network security group attached.",
						s.ID,
					)
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
			return api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidLinkedVNet,
				s.Path,
				"The provided subnet '%s' is not in a Succeeded state",
				s.ID,
			)
		}

		_, net, err := net.ParseCIDR(*ss.AddressPrefix)
		if err != nil {
			return err
		}

		ones, _ := net.Mask.Size()
		if ones > 27 {
			return api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidLinkedVNet,
				s.Path,
				"The provided subnet '%s' is invalid: must be /27 or larger.",
				s.ID,
			)
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

	return nil, api.NewCloudError(
		http.StatusBadRequest,
		api.CloudErrorCodeInvalidLinkedVNet,
		"",
		"The provided subnet '%s' could not be found.",
		subnetID,
	)
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
