package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/util/location"
	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/davecgh/go-spew/spew"

	"github.com/sirupsen/logrus"
)

func lb(ctx context.Context, log *logrus.Entry) error {

	authorizer, err := auth.NewAuthorizerFromCLIWithResource("https://management.azure.com/")
	if err != nil {
		return err
	}

	client := mgmtresources.NewProvidersClient("225e02bc-43d0-43d1-a01a-17e584a4ef69")
	client.Authorizer = authorizer

	networks, err := client.Get(ctx, "Microsoft.Network", "")
	if err != nil {
		return err
	}

	for _, rt := range *networks.ResourceTypes {
		if rt.ResourceType != nil && *rt.ResourceType == "loadBalancers" {
			spew.Dump(rt)
			if rt.ZoneMappings == nil{
				continue
			}
			for _, zoneMapping := range *rt.ZoneMappings {
				if location.Normalize(*zoneMapping.Location) == "eastus" {
					spew.Dump(zoneMapping)
				}

			}
		}
	}
	return nil
}
