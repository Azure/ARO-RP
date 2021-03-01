package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
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
	"github.com/Azure/ARO-RP/pkg/util/permissions"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

// Dynamic validate in the operator context.
type Dynamic interface {
	ValidateVnetPermissions(ctx context.Context) error
	ValidateRouteTablesPermissions(ctx context.Context) error
	ValidateCIDRRanges(ctx context.Context) error
	ValidateVnetLocation(ctx context.Context) error
	ValidateProviders(ctx context.Context) error
	// etc
	// does Quota code go in here too?
}

type dynamic struct {
	log             *logrus.Entry
	oc              *api.OpenShiftCluster
	vnetr           *azure.Resource
	masterSubnetID  string
	workerSubnetIDs []string

	code string
	typ  string

	permissions     authorization.PermissionsClient
	providers       features.ProvidersClient
	virtualNetworks virtualNetworksGetClient
}

func NewValidator(log *logrus.Entry, env env.Core, oc *api.OpenShiftCluster, masterSubnetID string, workerSubnetIDs []string, subscriptionID string, authorizer refreshable.Authorizer, code string, typ string) (*dynamic, error) {
	vnetID, _, err := subnet.Split(masterSubnetID)
	if err != nil {
		return nil, err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return nil, err
	}

	return &dynamic{
		log:             log,
		oc:              oc,
		vnetr:           &vnetr,
		masterSubnetID:  masterSubnetID,
		workerSubnetIDs: workerSubnetIDs,

		code: code,
		typ:  typ,

		permissions:     authorization.NewPermissionsClient(env.Environment(), subscriptionID, authorizer),
		providers:       features.NewProvidersClient(env.Environment(), subscriptionID, authorizer),
		virtualNetworks: newVirtualNetworksCache(network.NewVirtualNetworksClient(env.Environment(), subscriptionID, authorizer)),
	}, nil
}

func (dv *dynamic) ValidateVnetPermissions(ctx context.Context) error {
	dv.log.Printf("ValidateVnetPermissions (%s)", dv.typ)

	err := dv.validateActions(ctx, dv.vnetr, []string{
		"Microsoft.Network/virtualNetworks/join/action",
		"Microsoft.Network/virtualNetworks/read",
		"Microsoft.Network/virtualNetworks/write",
		"Microsoft.Network/virtualNetworks/subnets/join/action",
		"Microsoft.Network/virtualNetworks/subnets/read",
		"Microsoft.Network/virtualNetworks/subnets/write",
	})

	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, dv.code, "", "The %s does not have Network Contributor permission on vnet '%s'.", dv.typ, dv.vnetr)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet '%s' could not be found.", dv.vnetr)
	}
	return err
}

func (dv *dynamic) ValidateRouteTablesPermissions(ctx context.Context) error {
	vnet, err := dv.virtualNetworks.Get(ctx, dv.vnetr.ResourceGroup, dv.vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	m := map[string]string{}

	rtID, err := getRouteTableID(&vnet, "properties.masterProfile.subnetId", dv.masterSubnetID)
	if err != nil {
		return err
	}

	if rtID != "" {
		m[strings.ToLower(rtID)] = "properties.masterProfile.subnetId"
	}

	for i, s := range dv.workerSubnetIDs {
		path := fmt.Sprintf("properties.workerProfiles[%d].subnetId", i)

		rtID, err := getRouteTableID(&vnet, path, s)
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
		err := dv.validateRouteTablePermissions(ctx, rt, m[rt])
		if err != nil {
			return err
		}
	}

	return nil
}

func (dv *dynamic) validateRouteTablePermissions(ctx context.Context, rtID string, path string) error {
	dv.log.Printf("validateRouteTablePermissions(%s, %s)", dv.typ, path)

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
		return api.NewCloudError(http.StatusBadRequest, dv.code, "", "The %s does not have Network Contributor permission on route table '%s'.", dv.typ, rtID)
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

func (dv *dynamic) ValidateCIDRRanges(ctx context.Context) error {
	dv.log.Print("ValidateCIDRRanges")

	vnet, err := dv.virtualNetworks.Get(ctx, dv.vnetr.ResourceGroup, dv.vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	var subnets []string
	var CIDRArray []*net.IPNet

	// unique names of subnets from all node pools
	for i, subnet := range dv.oc.Properties.WorkerProfiles {
		exists := false
		for _, s := range subnets {
			if strings.EqualFold(strings.ToLower(subnet.SubnetID), strings.ToLower(s)) {
				exists = true
				break
			}
		}

		if !exists {
			subnets = append(subnets, subnet.SubnetID)
			path := fmt.Sprintf("properties.workerProfiles[%d].subnetId", i)
			c, err := dv.validateSubnet(ctx, &vnet, path, subnet.SubnetID)
			if err != nil {
				return err
			}

			CIDRArray = append(CIDRArray, c)
		}
	}

	masterCIDR, err := dv.validateSubnet(ctx, &vnet, "properties.MasterProfile.subnetId", dv.oc.Properties.MasterProfile.SubnetID)
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

func (dv *dynamic) ValidateVnetLocation(ctx context.Context) error {
	dv.log.Print("ValidateVnetLocation")

	vnet, err := dv.virtualNetworks.Get(ctx, dv.vnetr.ResourceGroup, dv.vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	if !strings.EqualFold(*vnet.Location, dv.oc.Location) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet location '%s' must match the cluster location '%s'.", *vnet.Location, dv.oc.Location)
	}

	return nil
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

	ones, _ := net.Mask.Size()
	if ones > 27 {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The provided subnet '%s' is invalid: must be /27 or larger.", subnetID)
	}

	return net, nil
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

func getRouteTableID(vnet *mgmtnetwork.VirtualNetwork, path, subnetID string) (string, error) {
	s := findSubnet(vnet, subnetID)
	if s == nil {
		return "", api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The subnet '%s' could not be found.", subnetID)
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
