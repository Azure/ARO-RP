package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
)

var (
	PROD_REGIONS = []string{
		"australiacentral",
		"australiacentral2",
		"australiaeast",
		"australiasoutheast",
		"brazilsouth",
		"brazilsoutheast",
		"canadacentral",
		"canadaeast",
		"centralindia",
		"centralus",
		"centraluseuap",
		"eastasia",
		"eastus",
		"eastus2",
		"eastus2euap",
		"francecentral",
		"germanywestcentral",
		"japaneast",
		"japanwest",
		"koreacentral",
		"northcentralus",
		"northeurope",
		"norwaywest",
		"norwayeast",
		"southafricanorth",
		"southcentralus",
		"southeastasia",
		"southindia",
		"swedencentral",
		"switzerlandnorth",
		"switzerlandwest",
		"uaenorth",
		"uaecentral",
		"uksouth",
		"ukwest",
		"westcentralus",
		"westeurope",
		"westus",
		"westus2",
		"westus3",
	}
)

type Region struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type RegionInfo struct {
	Regions []Region `json:"regions"`
}

func regionListFromRegions(regions []string) RegionInfo {
	final := &RegionInfo{
		Regions: make([]Region, 0, len(regions)),
	}

	for _, region := range regions {
		final.Regions = append(final.Regions, Region{
			Name: region,
			URL:  fmt.Sprintf("%s.admin.aro.azure.com", region),
		})
	}

	return *final
}

func (f *realFetcher) Regions(ctx context.Context) (RegionInfo, error) {
	// TODO: Add dynamic method of returning regions

	return regionListFromRegions(PROD_REGIONS), nil
}

func (c *client) Regions(ctx context.Context) (RegionInfo, error) {
	return c.fetcher.Regions(ctx)
}
