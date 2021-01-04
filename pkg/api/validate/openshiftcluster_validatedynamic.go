package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/apparentlymart/go-cidr/cidr"
	jwt "github.com/form3tech-oss/jwt-go"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	utilpermissions "github.com/Azure/ARO-RP/pkg/util/permissions"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

// OpenShiftClusterFullDynamicValidator is the dynamic validator interface
// for the RP execution contexts
type OpenShiftClusterFullDynamicValidator interface {
	Dynamic(context.Context) error
}

// OpenShiftClusterSlimDynamicValidator is the dynamic validator interface
// for the cluster execution context
type OpenShiftClusterSlimDynamicValidator interface {
	Dynamic(context.Context) error
}

// NewOpenShiftClusterFullDynamicValidator creates a new OpenShiftClusterFullDynamicValidator
func NewOpenShiftClusterFullDynamicValidator(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, subscriptionDoc *api.SubscriptionDocument, fpAuthorizer refreshable.Authorizer) OpenShiftClusterFullDynamicValidator {
	return &openShiftClusterFullDynamicValidator{
		env: env,

		oc:              oc,
		subscriptionDoc: subscriptionDoc,
		fpAuthorizer:    fpAuthorizer,

		log: log,
	}
}

// NewOpenShiftClusterSlimDynamicValidator creates a new OpenShiftClusterSlimDynamicValidator
func NewOpenShiftClusterSlimDynamicValidator(log *logrus.Entry, clientID string, clientSecret api.SecureString, subscriptionID string, tenantID string, clusterSpec *arov1alpha1.ClusterSpec) OpenShiftClusterSlimDynamicValidator {
	spp := &api.ServicePrincipalProfile{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
	return &openShiftClusterSlimDynamicValidator{
		log:            log,
		spp:            spp,
		tenantID:       tenantID,
		subscriptionID: subscriptionID,
		clusterSpec:    clusterSpec,
	}
}

// openShiftClusterFullDynamicValidator is dynamic validator used inside RP context
// and can operate within cluster and firstParty service principals.
// It includes openShiftClusterSlimDynamicValidatora
type openShiftClusterFullDynamicValidator struct {
	env env.Interface

	oc              *api.OpenShiftCluster
	subscriptionDoc *api.SubscriptionDocument
	fpAuthorizer    refreshable.Authorizer

	log *logrus.Entry
}

// openShiftClusterSlimDynamicValidator is dynamic validator used inside clusters
// context and can operate ONLY with cluster service principal scope
type openShiftClusterSlimDynamicValidator struct {
	log *logrus.Entry

	spp             *api.ServicePrincipalProfile
	clusterSpec     *arov1alpha1.ClusterSpec
	tenantID        string
	subscriptionID  string
	environmentName string
}

// Dynamic validates an OpenShift cluster in the context of cluster
func (dv *openShiftClusterSlimDynamicValidator) Dynamic(ctx context.Context) error {
	azureEnv, err := azure.EnvironmentFromName(dv.environmentName)
	if err != nil {
		return err
	}
	spAuthorizer, err := validateServicePrincipalProfile(ctx, dv.log, dv.spp, dv.tenantID, azureEnv.ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	spPermissions := authorization.NewPermissionsClient(&azureEnv, dv.subscriptionID, spAuthorizer)

	vnetr, err := azure.ParseResourceID(dv.clusterSpec.VNetID)
	if err != nil {
		return err
	}

	err = validateVnetPermissions(ctx, dv.log, spAuthorizer, spPermissions, dv.clusterSpec.VNetID, &vnetr, api.CloudErrorCodeInvalidServicePrincipalPermissions, "provided service principal")
	if err != nil {
		return err
	}

	return nil
}

// Dynamic validates an OpenShift cluster
func (dv *openShiftClusterFullDynamicValidator) Dynamic(ctx context.Context) error {
	// TODO: Dynamic() should work on an enriched oc (in update), and it
	// currently doesn't.  One sticking point is handling subnet overlap
	// calculations.

	r, err := azure.ParseResourceID(dv.oc.ID)
	if err != nil {
		return err
	}

	fpPermissions := authorization.NewPermissionsClient(dv.env.Environment(), dv.subscriptionDoc.ID, dv.fpAuthorizer)
	spAuthorizer, err := validateServicePrincipalProfile(ctx, dv.log, &dv.oc.Properties.ServicePrincipalProfile, dv.subscriptionDoc.Subscription.Properties.TenantID, dv.env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	spPermissions := authorization.NewPermissionsClient(dv.env.Environment(), r.SubscriptionID, spAuthorizer)
	spProviders := features.NewProvidersClient(dv.env.Environment(), r.SubscriptionID, spAuthorizer)
	spVirtualNetworks := network.NewVirtualNetworksClient(dv.env.Environment(), r.SubscriptionID, spAuthorizer)

	vnetID, _, err := subnet.Split(dv.oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	err = validateVnetPermissions(ctx, dv.log, spAuthorizer, spPermissions, vnetID, &vnetr, api.CloudErrorCodeInvalidServicePrincipalPermissions, "provided service principal")
	if err != nil {
		return err
	}

	err = validateVnetPermissions(ctx, dv.log, dv.fpAuthorizer, fpPermissions, vnetID, &vnetr, api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
	if err != nil {
		return err
	}

	// Get after validating permissions
	vnet, err := spVirtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	err = validateRouteTablePermissions(ctx, dv.log, dv.oc, spAuthorizer, spPermissions, &vnet, api.CloudErrorCodeInvalidServicePrincipalPermissions, "provided service principal")
	if err != nil {
		return err
	}

	err = validateRouteTablePermissions(ctx, dv.log, dv.oc, dv.fpAuthorizer, fpPermissions, &vnet, api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
	if err != nil {
		return err
	}

	err = validateVnet(ctx, dv.log, dv.oc, &vnet)
	if err != nil {
		return err
	}

	err = validateProviders(ctx, dv.log, spProviders)
	if err != nil {
		return err
	}

	return nil
}

type azureClaim struct {
	Roles []string `json:"roles,omitempty"`
}

func (*azureClaim) Valid() error {
	return fmt.Errorf("unimplemented")
}

func validateServicePrincipalProfile(ctx context.Context, log *logrus.Entry, spp *api.ServicePrincipalProfile, tenantID, resource string) (refreshable.Authorizer, error) {
	log.Print("validateServicePrincipalProfile")

	token, err := aad.GetToken(ctx, log, spp, tenantID, resource)
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

func validateVnetPermissions(ctx context.Context, log *logrus.Entry, authorizer refreshable.Authorizer, client authorization.PermissionsClient, vnetID string, vnetr *azure.Resource, code, typ string) error {
	log.Printf("validateVnetPermissions (%s)", typ)

	err := validateActions(ctx, log, vnetr, []string{
		"Microsoft.Network/virtualNetworks/join/action",
		"Microsoft.Network/virtualNetworks/read",
		"Microsoft.Network/virtualNetworks/write",
		"Microsoft.Network/virtualNetworks/subnets/join/action",
		"Microsoft.Network/virtualNetworks/subnets/read",
		"Microsoft.Network/virtualNetworks/subnets/write",
	}, authorizer, client)
	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, code, "", "The %s does not have Network Contributor permission on vnet '%s'.", typ, vnetID)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet '%s' could not be found.", vnetID)
	}
	return err
}

func validateRouteTablePermissions(ctx context.Context, log *logrus.Entry, oc *api.OpenShiftCluster, authorizer refreshable.Authorizer, client authorization.PermissionsClient, vnet *mgmtnetwork.VirtualNetwork, code, typ string) error {
	err := validateRouteTablePermissionsSubnet(ctx, log, authorizer, client, vnet, oc.Properties.MasterProfile.SubnetID, "properties.masterProfile.subnetId", code, typ)
	if err != nil {
		return err
	}

	for i, s := range oc.Properties.WorkerProfiles {
		err := validateRouteTablePermissionsSubnet(ctx, log, authorizer, client, vnet, s.SubnetID, "properties.workerProfiles["+strconv.Itoa(i)+"].subnetId", code, typ)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateRouteTablePermissionsSubnet(ctx context.Context, log *logrus.Entry, authorizer refreshable.Authorizer, client authorization.PermissionsClient, vnet *mgmtnetwork.VirtualNetwork, subnetID, path, code, typ string) error {
	log.Printf("validateRouteTablePermissionsSubnet(%s, %s)", typ, path)

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

	err = validateActions(ctx, log, &rtr, []string{
		"Microsoft.Network/routeTables/join/action",
		"Microsoft.Network/routeTables/read",
		"Microsoft.Network/routeTables/write",
	}, authorizer, client)
	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, code, "", "The %s does not have Network Contributor permission on route table '%s'.", typ, *s.RouteTable.ID)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedRouteTable, "", "The route table '%s' could not be found.", *s.RouteTable.ID)
	}
	return err
}

func validateSubnet(ctx context.Context, log *logrus.Entry, oc *api.OpenShiftCluster, vnet *mgmtnetwork.VirtualNetwork, path, subnetID string) (*net.IPNet, error) {
	log.Printf("validateSubnet (%s)", path)

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

	if strings.EqualFold(oc.Properties.MasterProfile.SubnetID, subnetID) {
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

	if oc.Properties.ProvisioningState == api.ProvisioningStateCreating {
		if s.SubnetPropertiesFormat != nil &&
			s.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided subnet '%s' is invalid: must not have a network security group attached.", subnetID)
		}

	} else {
		nsgID, err := subnet.NetworkSecurityGroupID(oc, *s.ID)
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
// and the subnets do not overlap with cluster pod/service CIDR blocks
func validateVnet(ctx context.Context, log *logrus.Entry, oc *api.OpenShiftCluster, vnet *mgmtnetwork.VirtualNetwork) error {
	log.Print("validateVnet")
	var err error

	if !strings.EqualFold(*vnet.Location, oc.Location) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet location '%s' must match the cluster location '%s'.", *vnet.Location, oc.Location)
	}

	// unique names of subnets from all node pools
	var subnets []string
	var CIDRArray []*net.IPNet
	for i, subnet := range oc.Properties.WorkerProfiles {
		exists := false
		for _, s := range subnets {
			if strings.EqualFold(strings.ToLower(subnet.SubnetID), strings.ToLower(s)) {
				exists = true
				break
			}
		}
		if !exists {
			subnets = append(subnets, subnet.SubnetID)
			c, err := validateSubnet(ctx, log, oc, vnet, "properties.workerProfiles["+strconv.Itoa(i)+"].subnetId", subnet.SubnetID)
			if err != nil {
				return err
			}
			CIDRArray = append(CIDRArray, c)
		}
	}
	masterSubnetCIDR, err := validateSubnet(ctx, log, oc, vnet, "properties.masterProfile.subnetId", oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	_, podCIDR, err := net.ParseCIDR(oc.Properties.NetworkProfile.PodCIDR)
	if err != nil {
		return err
	}

	_, serviceCIDR, err := net.ParseCIDR(oc.Properties.NetworkProfile.ServiceCIDR)
	if err != nil {
		return err
	}

	CIDRArray = append(CIDRArray, masterSubnetCIDR, podCIDR, serviceCIDR)

	err = cidr.VerifyNoOverlap(CIDRArray, &net.IPNet{IP: net.IPv4zero, Mask: net.IPMask(net.IPv4zero)})
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

func validateProviders(ctx context.Context, log *logrus.Entry, spProviders features.ProvidersClient) error {
	log.Print("validateProviders")

	providers, err := spProviders.List(ctx, nil, "")
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
			_, err = authorizer.RefreshWithContext(ctx, log)
			return false, err
		}
		if err != nil {
			return false, err
		}

		for _, action := range actions {
			ok, err := utilpermissions.CanDoAction(permissions, action)
			if !ok || err != nil {
				return false, err
			}
		}

		return true, nil
	}, timeoutCtx.Done())
}
