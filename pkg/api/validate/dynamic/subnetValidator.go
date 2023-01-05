package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate/dynamic/vnetcache"
	networkutil "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type SubnetValidator interface {
	Validate(ctx context.Context, oc *api.OpenShiftCluster, subnets []Subnet) error
}

type defaultSubnetValidator struct {
	log             *logrus.Entry
	virtualNetworks vnetcache.VirtualNetworksGetClient
}

func NewSubnetValidator(log *logrus.Entry, networkClient networkutil.VirtualNetworksClient) defaultSubnetValidator {
	vnetcache := vnetcache.NewVirtualNetworksCache(networkClient)
	return defaultSubnetValidator{log: log, virtualNetworks: vnetcache}
}

func (dv defaultSubnetValidator) Validate(ctx context.Context, oc *api.OpenShiftCluster, subnets []Subnet) error {
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

		vnet, err := dv.virtualNetworks.Get(ctx, vnetcache.CacheKeyFromResource(vnetr))
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
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, s.Path, "The provided subnet '%s' is invalid: must not have a network security group attached.", s.ID)
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
