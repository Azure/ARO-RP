package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
)

const (
	SharedMSIKeyVaultName       = "concat(take(resourceGroup().name,10), '" + SharedMSIKeyVaultNameSuffix + "')"
	SharedMSIKeyVaultNameSuffix = "-dev-msi"
)

// devMSIKeyvault returns an arm.Resource representing a shared key vault to be used for persisting mock MSI certificates when
// the RP is running in local development mode.
func (g *generator) devMSIKeyvault() *arm.Resource {
	return g.keyVault(fmt.Sprintf("[%s]", SharedMSIKeyVaultName), &[]mgmtkeyvault.AccessPolicyEntry{}, nil, nil)
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
