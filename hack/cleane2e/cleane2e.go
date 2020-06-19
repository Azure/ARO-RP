package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func main() {
	ctx := context.Background()
	log := utillog.GetLogger()

	if err := run(ctx, log); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, log *logrus.Entry) error {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	privatelinkservicescli := network.NewPrivateLinkServicesClient(subscriptionID, authorizer)
	resourcegroupscli := features.NewResourceGroupsClient(subscriptionID, authorizer)
	vnetscli := network.NewVirtualNetworksClient(subscriptionID, authorizer)

	gs, err := resourcegroupscli.List(ctx, "", nil)
	if err != nil {
		return err
	}

	sort.Slice(gs, func(i, j int) bool { return *gs[i].Name < *gs[j].Name })
	for _, g := range gs {
		if !strings.HasPrefix(*g.Name, "v4-e2e-rg-") &&
			!strings.HasPrefix(*g.Name, "aro-v4-e2e-rg-") {
			continue
		}

		if g.Tags["now"] == nil {
			continue
		}

		now, err := time.Parse(time.RFC3339Nano, *g.Tags["now"])
		if err != nil {
			log.Errorf("%s: %s", *g.Name, err)
			continue
		}
		if time.Now().Sub(now) < 6*time.Hour {
			continue
		}

		vnets, err := vnetscli.List(ctx, *g.Name)
		if err != nil {
			log.Errorf("%s: %s", *g.Name, err)
			continue
		}
		for _, vnet := range vnets {
			var changed bool

			for i := range *vnet.Subnets {
				(*vnet.Subnets)[i].NetworkSecurityGroup = nil
				changed = true
			}

			if changed {
				log.Printf("updating vnet %s/%s", *g.Name, *vnet.Name)
				vnetscli.CreateOrUpdate(ctx, *g.Name, *vnet.Name, vnet)
			}
		}

		plss, err := privatelinkservicescli.List(ctx, *g.Name)
		if err != nil {
			log.Errorf("%s: %s", *g.Name, err)
			continue
		}
		for _, pls := range plss {
			for _, peconn := range *pls.PrivateEndpointConnections {
				log.Printf("deleting private endpoint connection %s/%s/%s", *g.Name, *pls.Name, *peconn.Name)
				privatelinkservicescli.DeletePrivateEndpointConnection(ctx, *g.Name, *pls.Name, *peconn.Name)
			}
		}

		log.Printf("deleting resource group %s", *g.Name)
		resourcegroupscli.Delete(ctx, *g.Name)
	}

	return nil
}
