package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
)

var (
	// Storage accounts must not contain dashes or be more than 24 characters
	// Append "oic" to the pre-existing storage account prefix.
	storageAccountName         string = "concat(take(substring(parameters('storageAccountDomain'), 0, indexOf(parameters('storageAccountDomain'), '.')), 21), 'oic')"
	resourceTypeStorageAccount string = "Microsoft.Storage/storageAccounts"
)

func (g *generator) oicStorageAccount() *arm.Resource {
	storageAccount := &mgmtstorage.Account{
		Kind: mgmtstorage.StorageV2,
		Sku: &mgmtstorage.Sku{
			Name: "Standard_LRS",
		},
		AccountProperties: &mgmtstorage.AccountProperties{
			AllowBlobPublicAccess:  to.BoolPtr(true),
			EnableHTTPSTrafficOnly: to.BoolPtr(true),
			MinimumTLSVersion:      mgmtstorage.TLS12,
			AccessTier:             mgmtstorage.Hot,
		},
		Name:     to.StringPtr(fmt.Sprintf("[%s]", storageAccountName)),
		Location: to.StringPtr("[resourceGroup().location]"),
		Type:     to.StringPtr(resourceTypeStorageAccount),
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
