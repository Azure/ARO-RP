package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-09-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
)

var (
	storageAccountName         string = "parameters('oidcStorageAccountName')"
	resourceTypeStorageAccount string = "Microsoft.Storage/storageAccounts"
)

func (g *generator) oicStorageAccount() *arm.Resource {
	storageAccount := &mgmtstorage.Account{
		Kind: mgmtstorage.KindStorageV2,
		Sku: &mgmtstorage.Sku{
			Name: "Standard_LRS",
		},
		AccountProperties: &mgmtstorage.AccountProperties{
			AllowBlobPublicAccess:  to.BoolPtr(false),
			EnableHTTPSTrafficOnly: to.BoolPtr(true),
			MinimumTLSVersion:      mgmtstorage.MinimumTLSVersionTLS12,
			AccessTier:             mgmtstorage.AccessTierHot,
			AllowSharedKeyAccess:   to.BoolPtr(false),
			// Production has Public Network Access Disabled as OIDC Storage Account will be accessed via Azure Front Door
		},
		Name:     to.StringPtr(fmt.Sprintf("[%s]", storageAccountName)),
		Location: to.StringPtr("[resourceGroup().location]"),
		Type:     to.StringPtr(resourceTypeStorageAccount),
		Tags: map[string]*string{
			tagKeyExemptPublicBlob: to.StringPtr(tagValueExemptPublicBlob),
		},
	}

	return &arm.Resource{
		Resource:   storageAccount,
		APIVersion: azureclient.APIVersion("Microsoft.Storage"),
	}
}

func (g *generator) oicRoleAssignment() *arm.Resource {
	return rbac.ResourceRoleAssignmentWithName(
		rbac.RoleStorageBlobDataContributor,
		"parameters('rpServicePrincipalId')", // RP MSI
		resourceTypeStorageAccount,
		storageAccountName,
		fmt.Sprintf("concat(%s, '/Microsoft.Authorization/', guid(resourceId('%s', %s)))", storageAccountName, resourceTypeStorageAccount, storageAccountName),
	)
}
