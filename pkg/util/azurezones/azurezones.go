package azurezones

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/computeskus"
)

const (
	ALLOW_EXPANDED_AZ_ENV       = "ARO_INSTALLER_ALLOW_EXPANDED_AZS"
	CONTROL_PLANE_MACHINE_COUNT = 3
	WORKER_MINIMUM_ZONE_COUNT   = 3
)

// SupportedZones are the availability zones ARO deploys into. Zones outside
// this set are only used when expanded availability zones are enabled.
var SupportedZones = []string{"1", "2", "3"}

// Errors returned by DetermineAvailabilityZones. They carry the facts needed to
// describe the failure; the caller is responsible for wording and for mapping
// them to a customer-facing error.
var (
	// ErrMixedZonal is returned when exactly one of the control plane or worker
	// SKU is available in availability zones.
	ErrMixedZonal = errors.New("cluster creation with a mix of zonal and non-zonal SKUs is unsupported")
	// ErrInsufficientZones is returned when a zonal SKU is available in fewer
	// than the required number of supported availability zones.
	ErrInsufficientZones = errors.New("insufficient availability zones for SKU")
)

// ZoneError describes a failed availability zone check.
type ZoneError struct {
	// Err is ErrMixedZonal or ErrInsufficientZones.
	Err error
	// Profile is the VM profile the error applies to. For ErrMixedZonal it is
	// the profile whose SKU is not available in availability zones, i.e. the
	// one the customer has to change.
	Profile VMRole
	// SKUName is the name of the VM SKU for Profile.
	SKUName string
	// SKUZones is the number of availability zones the SKU offers in the
	// region, before ARO's supported zones are applied.
	SKUZones int
	// SupportedZones is the number of ARO-supported zones the SKU offers.
	SupportedZones int
	// RequiredZones is the number of zones the cluster needs.
	RequiredZones int
	// OtherSKUName is the name of the other profile's SKU. Only set for
	// ErrMixedZonal.
	OtherSKUName string
}

func (e *ZoneError) Error() string {
	if errors.Is(e.Err, ErrMixedZonal) {
		return fmt.Sprintf("%s: %s SKU %q", e.Err, e.Profile, e.SKUName)
	}
	return fmt.Sprintf("%s: %s SKU %q available in %d of %d zones (needs %d)",
		e.Err, e.Profile, e.SKUName, e.SupportedZones, e.SKUZones, e.RequiredZones)
}

func (e *ZoneError) Unwrap() error { return e.Err }

// VMRole identifies which VM profile a zone error applies to. The values are
// identifiers, not customer-facing text: wording belongs to the caller.
type VMRole string

const (
	VMRoleMaster VMRole = "master"
	VMRoleWorker VMRole = "worker"
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
		onlyBasicAZs := func(s string) bool {
			return !slices.Contains(SupportedZones, s)
		}
		return slices.DeleteFunc(zones, onlyBasicAZs)
	}

	return zones
}

func (m *availabilityZoneManager) DetermineAvailabilityZones(controlPlaneSKU, workerSKU *armcompute.ResourceSKU) ([]string, []string, []string, error) {
	controlPlaneZones := computeskus.Zones(controlPlaneSKU)
	workerZones := computeskus.Zones(workerSKU)

	// Whether a SKU is zonal is a property of the region, not of ARO's
	// supported zones, so record it before FilterZones can remove every zone.
	controlPlaneZonal := len(controlPlaneZones) > 0
	workerZonal := len(workerZones) > 0

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
	// A zonal SKU that FilterZones emptied has no ARO-supported zones, which is
	// an insufficient-zones failure, not a non-zonal success.
	if controlPlaneZonal && len(controlPlaneZones) < CONTROL_PLANE_MACHINE_COUNT {
		return nil, nil, nil, &ZoneError{
			Err: ErrInsufficientZones, Profile: VMRoleMaster, SKUName: *controlPlaneSKU.Name,
			SKUZones: len(computeskus.Zones(controlPlaneSKU)), SupportedZones: len(controlPlaneZones),
			RequiredZones: CONTROL_PLANE_MACHINE_COUNT,
		}
	} else if len(controlPlaneZones) == 0 {
		controlPlaneZones = []string{}
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
	if workerZonal && len(workerZones) < WORKER_MINIMUM_ZONE_COUNT {
		return nil, nil, nil, &ZoneError{
			Err: ErrInsufficientZones, Profile: VMRoleWorker, SKUName: *workerSKU.Name,
			SKUZones: len(computeskus.Zones(workerSKU)), SupportedZones: len(workerZones),
			RequiredZones: WORKER_MINIMUM_ZONE_COUNT,
		}
	} else if len(workerZones) == 0 {
		workerZones = []string{}
	}

	// Mixed zonal/non-zonal is checked after the sufficiency checks: at this
	// point a side with zones has at least the required number of supported
	// zones, so the only way the two sides can differ is when one of them is
	// genuinely non-zonal. Deciding this on pre-filter zonal-ness earlier would
	// reject a zonal control plane that only offers unsupported zones (e.g.
	// 4,5,6) paired with a non-zonal worker, which used to deploy as a
	// non-zonal cluster.
	if controlPlaneZonal != workerZonal {
		if !controlPlaneZonal {
			return nil, nil, nil, &ZoneError{
				Err: ErrMixedZonal, Profile: VMRoleMaster, SKUName: *controlPlaneSKU.Name,
				SKUZones: len(computeskus.Zones(controlPlaneSKU)), OtherSKUName: *workerSKU.Name,
			}
		}
		return nil, nil, nil, &ZoneError{
			Err: ErrMixedZonal, Profile: VMRoleWorker, SKUName: *workerSKU.Name,
			SKUZones: len(computeskus.Zones(workerSKU)), OtherSKUName: *controlPlaneSKU.Name,
		}
	}

	return controlPlaneZones, workerZones, allAvailableZones, nil
}
