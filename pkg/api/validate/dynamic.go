package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	utilpermissions "github.com/Azure/ARO-RP/pkg/util/permissions"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type SlimDynamic interface {
	ValidateVnetPermissions(ctx context.Context, code, typ string) error
	// etc
	// does Quota code go in here too?
}

type dynamic struct {
	log   *logrus.Entry
	oc    *api.OpenShiftCluster
	vnetr *azure.Resource

	permissions     authorization.PermissionsClient
	providers       features.ProvidersClient
	virtualNetworks virtualNetworksGetClient
}

// TODO: get rid of subscriptionDoc here
func NewValidator(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, subscriptionDoc *api.SubscriptionDocument, authorizer refreshable.Authorizer) (*dynamic, error) {
	vnetID, _, err := subnet.Split(oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return nil, err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return nil, err
	}

	return &dynamic{
		log:   log,
		oc:    oc,
		vnetr: &vnetr,

		permissions:     authorization.NewPermissionsClient(env.Environment(), subscriptionDoc.ID, authorizer),
		providers:       features.NewProvidersClient(env.Environment(), subscriptionDoc.ID, authorizer),
		virtualNetworks: newVirtualNetworksCache(network.NewVirtualNetworksClient(env.Environment(), subscriptionDoc.ID, authorizer)),
	}, nil
}

/*
Dynamic() {
	get fp authorizer
	create a dynamic{fpauthorizer}
	validateVnetPermissions
	if err { bail }
	validateRouteTablesPermissions
	if err { bail }

	get sp authorizer
	create a dynamic{spauthorizer}
	validateVnetPermissions
	if err { bail }
	validateRouteTablesPermissions
	if err { bail }

	do all the other checks
	validateVnet
	if err { bail }
	etc
}

operator context {
	get sp authorizer
	create a dynamic{spauthorizer}
	pick and choose...
	validateVnetPermissions
	if err { note error and continue }
	validateRouteTablesPermissions
	if err { note error and continue }
}

*/

// Dynamic validates an OpenShift cluster
func (dv *dynamic) Dynamic(ctx context.Context) error {
	err := dv.validateVnetPermissions(ctx, api.CloudErrorCodeInvalidServicePrincipalPermissions, "provided service principal")
	if err != nil {
		return err
	}

	// after validating vnet permissions
	err = dv.validateRouteTablesPermissions(ctx, api.CloudErrorCodeInvalidServicePrincipalPermissions, "provided service principal")
	if err != nil {
		return err
	}

	err = dv.validateVnet(ctx)
	if err != nil {
		return err
	}

	return dv.validateProviders(ctx)
}

func (dv *dynamic) validateVnetPermissions(ctx context.Context, code, typ string) error {
	dv.log.Printf("validateVnetPermissions (%s)", typ)

	err := dv.validateActions(ctx, dv.vnetr, []string{
		"Microsoft.Network/virtualNetworks/join/action",
		"Microsoft.Network/virtualNetworks/read",
		"Microsoft.Network/virtualNetworks/write",
		"Microsoft.Network/virtualNetworks/subnets/join/action",
		"Microsoft.Network/virtualNetworks/subnets/read",
		"Microsoft.Network/virtualNetworks/subnets/write",
	})
	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, code, "", "The %s does not have Network Contributor permission on vnet '%s'.", typ, dv.vnetr)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet '%s' could not be found.", dv.vnetr)
	}
	return err
}

func (dv *dynamic) validateRouteTablesPermissions(ctx context.Context, code, typ string) error {
	vnet, err := dv.virtualNetworks.Get(ctx, dv.vnetr.ResourceGroup, dv.vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	m := map[string]string{}

	rtID, err := getRouteTableID(&vnet, "properties.masterProfile.subnetId", dv.oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	if rtID != "" {
		m[strings.ToLower(rtID)] = "properties.masterProfile.subnetId"
	}

	for i, s := range dv.oc.Properties.WorkerProfiles {
		path := fmt.Sprintf("properties.workerProfiles[%d].subnetId", i)

		rtID, err := getRouteTableID(&vnet, path, s.SubnetID)
		if err != nil {
			return err
		}

		if _, ok := m[strings.ToLower(rtID)]; ok || rtID == "" {
			continue
		}

		m[strings.ToLower(rtID)] = path
	}

	rts := make([]string, 0, len(m))
	for rt := range m {
		rts = append(rts, rt)
	}

	sort.Slice(rts, func(i, j int) bool { return strings.Compare(m[rts[i]], m[rts[j]]) < 0 })

	for _, rt := range rts {
		err := dv.validateRouteTablePermissions(ctx, rt, m[rt], code, typ)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dv *dynamic) validateRouteTablePermissions(ctx context.Context, rtID, path, code, typ string) error {
	dv.log.Printf("validateRouteTablePermissions(%s, %s)", typ, path)

	rtr, err := azure.ParseResourceID(rtID)
	if err != nil {
		return err
	}

	err = dv.validateActions(ctx, &rtr, []string{
		"Microsoft.Network/routeTables/join/action",
		"Microsoft.Network/routeTables/read",
		"Microsoft.Network/routeTables/write",
	})
	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, code, "", "The %s does not have Network Contributor permission on route table '%s'.", typ, rtID)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedRouteTable, "", "The route table '%s' could not be found.", rtID)
	}
	return err
}

func (dv *dynamic) validateSubnet(ctx context.Context, vnet *mgmtnetwork.VirtualNetwork, path, subnetID string) (*net.IPNet, error) {
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

// validateVnet checks that the vnet does not have custom dns servers set
// and the subnets do not overlap with cluster pod/service CIDR blocks
func (dv *dynamic) validateVnet(ctx context.Context) error {
	dv.log.Print("validateVnet")

	vnet, err := dv.virtualNetworks.Get(ctx, dv.vnetr.ResourceGroup, dv.vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	if !strings.EqualFold(*vnet.Location, dv.oc.Location) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet location '%s' must match the cluster location '%s'.", *vnet.Location, dv.oc.Location)
	}

	// unique names of subnets from all node pools
	subnets := map[string]struct{}{}
	var CIDRArray []*net.IPNet
	for i, subnet := range dv.oc.Properties.WorkerProfiles {
		if _, ok := subnets[strings.ToLower(subnet.SubnetID)]; ok {
			continue
		}

		subnets[strings.ToLower(subnet.SubnetID)] = struct{}{}

		c, err := dv.validateSubnet(ctx, &vnet, "properties.workerProfiles["+strconv.Itoa(i)+"].subnetId", subnet.SubnetID)
		if err != nil {
			return err
		}
		CIDRArray = append(CIDRArray, c)
	}
	masterSubnetCIDR, err := dv.validateSubnet(ctx, &vnet, "properties.masterProfile.subnetId", dv.oc.Properties.MasterProfile.SubnetID)
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

func (dv *dynamic) validateProviders(ctx context.Context) error {
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

func (dv *dynamic) validateActions(ctx context.Context, r *azure.Resource, actions []string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		permissions, err := dv.permissions.ListForResource(ctx, r.ResourceGroup, r.Provider, "", r.ResourceType, r.ResourceName)
		if detailedErr, ok := err.(autorest.DetailedError); ok &&
			detailedErr.StatusCode == http.StatusForbidden {
			return false, steps.ErrWantRefresh
		}
		if err != nil {
			return false, err
		}

		for _, action := range actions {
			ok, err := utilpermissions.CanDoAction(permissions, action)
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
