package zones

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

type Interface interface {
	Zones(vmSize string) ([]string, error)
}

type zones struct {
	zones map[string][]string
}

func NewZones(ctx context.Context, im instancemetadata.InstanceMetadata) (Interface, error) {
	rpAuthorizer, err := env.RPAuthorizer(azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	c := compute.NewResourceSkusClient(im.SubscriptionID(), rpAuthorizer)

	skus, err := c.List(ctx, "")
	if err != nil {
		return nil, err
	}

	m := map[string][]string{}

	for _, sku := range skus {
		if !strings.EqualFold((*sku.Locations)[0], im.Location()) ||
			*sku.ResourceType != "virtualMachines" {
			continue
		}

		m[*sku.Name] = *(*sku.LocationInfo)[0].Zones
	}

	return &zones{
		zones: m,
	}, nil
}

func (z *zones) Zones(vmSize string) ([]string, error) {
	zones, found := z.zones[vmSize]
	if !found {
		return nil, fmt.Errorf("zone information not found for vm size %q", vmSize)
	}
	return zones, nil
}
