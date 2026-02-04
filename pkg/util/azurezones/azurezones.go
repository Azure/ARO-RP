package azurezones

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"slices"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/computeskus"
)

const (
	ALLOW_EXPANDED_AZ_ENV       = "ARO_INSTALLER_ALLOW_EXPANDED_AZS"
	CONTROL_PLANE_MACHINE_COUNT = 3
)

type availabilityZoneManager struct {
	allowExpandedAvailabilityZones bool
}

func NewManager(allowExpandedAvailabilityZones bool) *availabilityZoneManager {
	return &availabilityZoneManager{
		allowExpandedAvailabilityZones: allowExpandedAvailabilityZones,
	}
}

func (m *availabilityZoneManager) FilterZones(zones []string) []string {
	// Gate allowing expanded AZs behind feature flag
	if !m.allowExpandedAvailabilityZones {
		basicAZs := []string{"1", "2", "3"}
		onlyBasicAZs := func(s string) bool {
			return !slices.Contains(basicAZs, s)
		}
		return slices.DeleteFunc(zones, onlyBasicAZs)
	}

	return zones
}

func (m *availabilityZoneManager) DetermineAvailabilityZones(controlPlaneSKU, workerSKU *armcompute.ResourceSKU) ([]string, []string, []string, error) {
	controlPlaneZones := computeskus.Zones(controlPlaneSKU)
	workerZones := computeskus.Zones(workerSKU)

	// We sort the zones so that we will pick them in numerical order if we need
	// less replicas than zones. With non-basic AZs, this means that control
	// plane nodes will not go onto the 4th AZ by default. For workers, if more
	// than 3 are specified on cluster creation, they will be spread across all
	// available zones, but will pick 1,2,3 in the normal 3-node configuration.
	// This is likely less surprising for setups where a 4th AZ might cause
	// automation to fail by picking, e.g. zones 1, 2, 4. We may wish to be
	// smarter about this in future. Note: If expanded AZs are available (see
	// the env var) and a SKU is available in e.g. zones 1, 2, 4, we will deploy
	// control planes there.
	slices.Sort(controlPlaneZones)
	slices.Sort(workerZones)

	controlPlaneZones = m.FilterZones(controlPlaneZones)
	workerZones = m.FilterZones(workerZones)

	if (len(controlPlaneZones) == 0 && len(workerZones) > 0) ||
		(len(workerZones) == 0 && len(controlPlaneZones) > 0) {
		return nil, nil, nil, fmt.Errorf("cluster creation with mix of zonal and non-zonal resources is unsupported (control plane zones: %d, worker zones: %d)", len(controlPlaneZones), len(workerZones))
	}

	// Once we've removed the expanded AZs (if applicable), get the super-set of
	// AZs for deploying PIPs/Frontend IPs in
	allAvailableZones := slices.Concat(controlPlaneZones, workerZones)
	slices.Sort(allAvailableZones)
	allAvailableZones = slices.Compact(allAvailableZones)
	if len(allAvailableZones) == 0 {
		allAvailableZones = []string{}
	}

	// We handle the case where regions have no zones or >= zones than replicas,
	// but not when replicas > zones. We (currently) only support 3 control
	// plane replicas and Azure AZs will always be a minimum of 3, see
	// https://azure.microsoft.com/en-us/blog/our-commitment-to-expand-azure-availability-zones-to-more-regions/
	if len(controlPlaneZones) == 0 {
		controlPlaneZones = []string{}
	} else if len(controlPlaneZones) < CONTROL_PLANE_MACHINE_COUNT {
		return nil, nil, nil, fmt.Errorf("control plane SKU '%s' only available in %d zones, need %d", *controlPlaneSKU.Name, len(controlPlaneZones), CONTROL_PLANE_MACHINE_COUNT)
	} else if len(controlPlaneZones) >= CONTROL_PLANE_MACHINE_COUNT {
		// Pick lower zones first
		controlPlaneZones = controlPlaneZones[:CONTROL_PLANE_MACHINE_COUNT]
	}

	// Unlike above, we don't particularly mind if we pass the Installer more
	// zones than the usual 3 in a zonal region, since it automatically balances
	// them across the available zones. However, if a SKU is available in less
	// than 3 regions we will fail, since taints on cluster components like
	// Prometheus may prevent the eventual install from turning healthy. As
	// such, prevent situations where 2 workers may be deployed on one zone and
	// 1 on another, even though OpenShift treats that as a theoretically valid
	// configuration.
	if len(workerZones) == 0 {
		workerZones = []string{}
	} else if len(workerZones) < 3 {
		return nil, nil, nil, fmt.Errorf("worker SKU '%s' only available in %d zones, need %d", *workerSKU.Name, len(workerZones), 3)
	}

	return controlPlaneZones, workerZones, allAvailableZones, nil
}
