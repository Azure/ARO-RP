package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	utilpermissions "github.com/Azure/ARO-RP/pkg/util/permissions"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

// SlimDynamic validate in the operator context.
type SlimDynamic interface {
	ValidateVnetPermissions(ctx context.Context, code string, typ string) error
	ValidateRouteTablesPermissions(ctx context.Context, code string, typ string) error
	ValidateVnetDNS(ctx context.Context) error
	// etc
	// does Quota code go in here too?
}

type dynamic struct {
	log             *logrus.Entry
	vnetr           *azure.Resource
	masterSubnetID  string
	workerSubnetIDs []string

	permissions     authorization.PermissionsClient
	virtualNetworks virtualNetworksGetClient
}

func NewValidator(log *logrus.Entry, azEnv *azure.Environment, masterSubnetID string, workerSubnetIDs []string, subscriptionID string, authorizer refreshable.Authorizer) (*dynamic, error) {
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
		vnetr:           &vnetr,
		masterSubnetID:  masterSubnetID,
		workerSubnetIDs: workerSubnetIDs,

		permissions:     authorization.NewPermissionsClient(azEnv, subscriptionID, authorizer),
		virtualNetworks: newVirtualNetworksCache(network.NewVirtualNetworksClient(azEnv, subscriptionID, authorizer)),
	}, nil
}

func (dv *dynamic) ValidateVnetPermissions(ctx context.Context, code string, typ string) error {
	dv.log.Printf("ValidateVnetPermissions (%s)", typ)

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

func (dv *dynamic) ValidateRouteTablesPermissions(ctx context.Context, code string, typ string) error {
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
		err := dv.validateRouteTablePermissions(ctx, rt, m[rt], code, typ)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dv *dynamic) validateRouteTablePermissions(ctx context.Context, rtID string, path string, code string, typ string) error {
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

func (dv *dynamic) ValidateVnetDNS(ctx context.Context) error {
	dv.log.Print("validateVnetDns")

	vnet, err := dv.virtualNetworks.Get(ctx, dv.vnetr.ResourceGroup, dv.vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	if vnet.DhcpOptions != nil &&
		vnet.DhcpOptions.DNSServers != nil &&
		len(*vnet.DhcpOptions.DNSServers) > 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided vnet '%s' is invalid: custom DNS servers are not supported.", *vnet.ID)
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
