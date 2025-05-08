package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
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
		"qatar",
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

func (p *portal) regions(w http.ResponseWriter, r *http.Request) {
	resp := &RegionInfo{
		Regions: make([]Region, 0, len(PROD_REGIONS)),
	}

	if value, found := os.LookupEnv("AZURE_ENVIRONMENT"); found {
		// AZURE_ENVIRONMENT variable can either be AZUREPUBLICCLOUD or AZUREUSGOVERNMENTCLOUD
		if strings.EqualFold(value, azureclient.PublicCloud.Environment.Name) {
			for _, region := range PROD_REGIONS {
				resp.Regions = append(resp.Regions, Region{
					Name: region,
					URL:  fmt.Sprintf("https://%s.admin.aro.azure.com", region),
				})
			}
		}
	}

	b, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
