package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	"github.com/Azure/ARO-RP/pkg/util/rpauthorizer"
)

func Find(ctx context.Context, instancemetadata instancemetadata.InstanceMetadata, rpauthorizer rpauthorizer.RPAuthorizer, tagValue string) (string, error) {
	rpAuthorizer, err := rpauthorizer.NewRPAuthorizer(azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return "", err
	}

	vaults := keyvault.NewVaultsClient(instancemetadata.SubscriptionID(), rpAuthorizer)

	vs, err := vaults.ListByResourceGroup(ctx, instancemetadata.ResourceGroup(), nil)
	if err != nil {
		return "", err
	}

	for _, v := range vs {
		if v.Tags[generator.KeyVaultTagName] != nil &&
			*v.Tags[generator.KeyVaultTagName] == tagValue {
			return *v.Properties.VaultURI, nil
		}
	}

	return "", fmt.Errorf("key vault with tag %s=%s not found", generator.KeyVaultTagName, tagValue)
}
