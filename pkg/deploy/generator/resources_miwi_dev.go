package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-09-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
)

var (
	storageAccountName         string = "parameters('oidcStorageAccountName')"
	resourceTypeStorageAccount string = "Microsoft.Storage/storageAccounts"
	resourceTypeBlobContainer  string = "blobServices/containers"

	SharedMSIKeyVaultName = "concat(take(resourceGroup().name,10), '" + env.SharedMSIKeyVaultNameSuffix + "')"
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

// devMSIKeyvault returns an arm.Resource representing a shared key vault to be used for persisting mock MSI certificates when
// the RP is running in local development mode.
func (g *generator) devMSIKeyvault() *arm.Resource {
	return g.keyVault(fmt.Sprintf("[%s]", SharedMSIKeyVaultName), &[]mgmtkeyvault.AccessPolicyEntry{}, nil, true, nil)
}

// devMSIKeyvaultRBAC returns an arm.Resource representing a role assignment that grants the local development mode's mock RP identity
// the KeyVaultSecretsOfficer role on the shared dev MSI key vault.
func (g *generator) devMSIKeyvaultRBAC() *arm.Resource {
	return rbac.ResourceRoleAssignment(
		rbac.RoleKeyVaultSecretsOfficer,
		"parameters('rpServicePrincipalId')",
		"Microsoft.KeyVault/vaults",
		SharedMSIKeyVaultName,
	)
}
