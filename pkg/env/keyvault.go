package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

const (
	KeyVaultTagName = "vault"
)

func GetVaultURI(ctx context.Context, im instancemetadata.InstanceMetadata, tag string) (string, error) {
	rpAuthorizer, err := RPAuthorizer(azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return "", err
	}

	vaults := keyvault.NewVaultsClient(im.SubscriptionID(), rpAuthorizer)

	vs, err := vaults.ListByResourceGroup(ctx, im.ResourceGroup(), nil)
	if err != nil {
		return "", err
	}

	var count int
	var uri string
	for _, v := range vs {
		if v.Tags[KeyVaultTagName] != nil &&
			*v.Tags[KeyVaultTagName] == tag {
			uri = *v.Properties.VaultURI
			count++
		}
	}

	if count != 1 {
		return "", fmt.Errorf("found %d key vaults with vault tag value %s, expected 1", count, tag)
	}

	return uri, nil
}
