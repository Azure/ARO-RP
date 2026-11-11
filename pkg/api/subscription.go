package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "fmt"

// Subscription represents a subscription
type Subscription struct {
	MissingFields

	State SubscriptionState `json:"state,omitempty"`

	Properties *SubscriptionProperties `json:"properties,omitempty"`
}

// SubscriptionState represents a subscription state
type SubscriptionState string

// SubscriptionState constants
const (
	SubscriptionStateRegistered   SubscriptionState = "Registered"
	SubscriptionStateUnregistered SubscriptionState = "Unregistered"
	SubscriptionStateWarned       SubscriptionState = "Warned"
	SubscriptionStateSuspended    SubscriptionState = "Suspended"
	SubscriptionStateDeleted      SubscriptionState = "Deleted"
)

// SubscriptionProperties represents subscription properties
type SubscriptionProperties struct {
	MissingFields

	TenantID           string                     `json:"tenantId,omitempty"`
	AccountOwner       *AccountOwnerProfile       `json:"accountOwner,omitempty"`
	RegisteredFeatures []RegisteredFeatureProfile `json:"registeredFeatures,omitempty"`
	AvailabilityZones  *AvailabilityZonesProfile  `json:"availabilityZones,omitempty"`
}

func (f *SubscriptionProperties) MapPhysicalZonesToLogicalZonesForSubscription(location string, physicalZones []string) ([]string, error) {
	// zero-length (i.e. non-zonal) does not need mapping, exit early
	if len(physicalZones) == 0 {
		return []string{}, nil
	}

	// No mapping, but non-zonal, just assume the regular zones -- 1..n for n
	// zones (e.g. 1,2,3 for 3-zones)
	if f.AvailabilityZones == nil {
		logicalZones := []string{}

		for zoneNum := range physicalZones {
			logicalZones = append(logicalZones, fmt.Sprintf("%d", zoneNum+1))
		}

		return logicalZones, nil
	}

	// for _, zoneProfile := range f.AvailabilityZones {
	// 	if zoneProfile.Location == location {
	logicalZones := []string{}

	for _, physicalZone := range physicalZones {
		foundLogicalZone := false

		// for _, zoneMapping := range zoneProfile.ZoneMappings {
		for _, zoneMapping := range f.AvailabilityZones.ZoneMappings {
			if physicalZone == zoneMapping.PhysicalZone {
				logicalZones = append(logicalZones, zoneMapping.LogicalZone)
				foundLogicalZone = true
				break
			}
		}

		if !foundLogicalZone {
			return nil, fmt.Errorf("unable to map physical zone '%s' to logical zone", physicalZone)
		}
	}

	return logicalZones, nil
	// 	}
	// }

	return nil, fmt.Errorf("unable to find location '%s' in subscription zone mappings", location)
}

// AccountOwnerProfile represents the subscription account owner information
type AccountOwnerProfile struct {
	MissingFields

	Email string `json:"email,omitempty"`
}

// RegisteredFeatureProfile represents the features registered to the subscription
type RegisteredFeatureProfile struct {
	MissingFields

	Name  string `json:"name,omitempty"`
	State string `json:"state,omitempty"`
}

// AvailabilityZonesProfile represents a location and the Availability Zone
// mappings in that location
type AvailabilityZonesProfile struct {
	Location     string                    `json:"location,omitempty"`
	ZoneMappings []AvailabilityZoneMapping `json:"zoneMappings,omitempty"`
}

// AvailabilityZoneMapping represents the relation between a Logical
// Availability Zone and a Physical Availability Zone
type AvailabilityZoneMapping struct {
	LogicalZone  string `json:"logicalZone,omitempty"`
	PhysicalZone string `json:"physicalZone,omitempty"`
}
