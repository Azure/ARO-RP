package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"encoding/pem"
	"os"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/version"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
)

func adminKeyvaultAccessPolicy(_env env.Core) map[string]interface{} {
	return map[string]interface{}{
		"objectId": os.Getenv("ADMIN_OBJECT_ID"),
		"permissions": map[string]interface{}{
			"certificates": []interface{}{
				"Get",
				"List",
				"Update",
				"Create",
				"Import",
				"Delete",
				"Recover",
				"Backup",
				"Restore",
				"ManageContacts",
				"ManageIssuers",
				"GetIssuers",
				"ListIssuers",
			},
			"secrets": []interface{}{
				"Get",
				"List",
			},
		},
		"tenantId": _env.TenantID(),
	}
}

func deployKeyvaultAccessPolicy(_env env.Core) map[string]interface{} {
	return map[string]interface{}{
		"objectId": os.Getenv("AZURE_SERVICE_PRINCIPAL_ID"),
		"permissions": map[string]interface{}{
			"secrets": []interface{}{
				"Get",
				"List",
				"Set",
			},
		},
		"tenantId": _env.TenantID(),
	}
}

func DevConfig(_env env.Core) (*Config, error) {
	ca, err := os.ReadFile("secrets/dev-ca.crt")
	if err != nil {
		return nil, err
	}

	client, err := os.ReadFile("secrets/dev-client.crt")
	if err != nil {
		return nil, err
	}

	clientCert, err := x509.ParseCertificate(client)
	if err != nil {
		return nil, err
	}

	sshPublicKeyPath := os.Getenv("SSH_PUBLIC_KEY")
	if sshPublicKeyPath == "" {
		sshPublicKeyPath = os.Getenv("HOME") + "/.ssh/id_rsa.pub"
	}

	sshPublicKey, err := os.ReadFile(sshPublicKeyPath)
	if err != nil {
		return nil, err
	}

	// use unique prefix for Azure resources when it is set, otherwise use your user's name
	azureUniquePrefix := os.Getenv("AZURE_PREFIX")
	if azureUniquePrefix == "" {
		azureUniquePrefix = os.Getenv("USER")
	}

	keyvaultPrefix := azureUniquePrefix + "-aro-" + _env.Location()
	if len(keyvaultPrefix) > 20 {
		keyvaultPrefix = keyvaultPrefix[:20]
	}

	oidcStorageAccountName := azureUniquePrefix + _env.Location()
	if len(oidcStorageAccountName) >= 21 {
		oidcStorageAccountName = oidcStorageAccountName[:21]
	}
	oidcStorageAccountName = oidcStorageAccountName + "oic"

	return &Config{
		RPs: []RPConfig{
			{
				Location:                 _env.Location(),
				SubscriptionID:           _env.SubscriptionID(),
				GatewayResourceGroupName: azureUniquePrefix + "-gwy-" + _env.Location(),
				RPResourceGroupName:      azureUniquePrefix + "-aro-" + _env.Location(),
				Configuration: &Configuration{
					AzureCloudName:         &_env.Environment().ActualCloudName,
					DatabaseAccountName:    to.Ptr(azureUniquePrefix + "-aro-" + _env.Location()),
					KeyvaultDNSSuffix:      &_env.Environment().KeyVaultDNSSuffix,
					KeyvaultPrefix:         &keyvaultPrefix,
					OIDCStorageAccountName: to.Ptr(oidcStorageAccountName),
					OtelAuditQueueSize:     to.Ptr("0"),
				},
			},
		},
		Configuration: &Configuration{
			ACRResourceID:                to.Ptr("/subscriptions/" + _env.SubscriptionID() + "/resourceGroups/" + azureUniquePrefix + "-global/providers/Microsoft.ContainerRegistry/registries/" + azureUniquePrefix + "aro"),
			AdminAPICABundle:             to.Ptr(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca}))),
			AdminAPIClientCertCommonName: &clientCert.Subject.CommonName,
			ARMAPICABundle:               to.Ptr(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca}))),
			ARMAPIClientCertCommonName:   &clientCert.Subject.CommonName,
			ARMClientID:                  to.Ptr(os.Getenv("AZURE_ARM_CLIENT_ID")),
			AzureSecPackVSATenantId:      to.Ptr(""),
			ClusterMDMAccount:            to.Ptr(version.DevClusterGenevaMetricsAccount),
			ClusterMDSDAccount:           to.Ptr(version.DevClusterGenevaLoggingAccount),
			ClusterMDSDConfigVersion:     to.Ptr(version.DevClusterGenevaLoggingConfigVersion),
			ClusterMDSDNamespace:         to.Ptr(version.DevClusterGenevaLoggingNamespace),
			ClusterParentDomainName:      to.Ptr(azureUniquePrefix + "-clusters." + os.Getenv("PARENT_DOMAIN_NAME")),
			CosmosDB: &CosmosDBConfiguration{
				StandardProvisionedThroughput: 1000,
				PortalProvisionedThroughput:   400,
				GatewayProvisionedThroughput:  400,
			},
			DisableCosmosDBFirewall: to.Ptr(true),
			ExtraClusterKeyvaultAccessPolicies: []interface{}{
				adminKeyvaultAccessPolicy(_env),
			},
			ExtraGatewayKeyvaultAccessPolicies: []interface{}{
				adminKeyvaultAccessPolicy(_env),
			},
			ExtraPortalKeyvaultAccessPolicies: []interface{}{
				adminKeyvaultAccessPolicy(_env),
				deployKeyvaultAccessPolicy(_env),
			},
			ExtraServiceKeyvaultAccessPolicies: []interface{}{
				adminKeyvaultAccessPolicy(_env),
				deployKeyvaultAccessPolicy(_env),
			},
			FluentbitImage:       to.Ptr(version.FluentbitImage(azureUniquePrefix + "aro." + _env.Environment().ContainerRegistryDNSSuffix)),
			FPClientID:           to.Ptr(os.Getenv("AZURE_FP_CLIENT_ID")),
			FPServicePrincipalID: to.Ptr(os.Getenv("AZURE_FP_SERVICE_PRINCIPAL_ID")),
			FPTenantID:           to.Ptr(os.Getenv("AZURE_TENANT_ID")),
			GatewayDomains: []string{
				"eastus-shared.ppe.warm.ingest.monitor.core.windows.net",
				"gcs.ppe.monitoring.core.windows.net",
				"gsm1890023205eh.servicebus.windows.net",
				"gsm1890023205xt.blob.core.windows.net",
				"gsm584263398eh.servicebus.windows.net",
				"gsm584263398xt.blob.core.windows.net",
				"gsm779889026eh.servicebus.windows.net",
				"gsm779889026xt.blob.core.windows.net",
				"monitoringagentbvt2.blob.core.windows.net",
				"qos.ppe.warm.ingest.monitor.core.windows.net",
				"test1.diagnostics.monitoring.core.windows.net",
			},
			GatewayMDSDConfigVersion:    to.Ptr(version.DevGatewayGenevaLoggingConfigVersion),
			GatewayVMSSCapacity:         to.Ptr(1),
			GlobalResourceGroupLocation: to.Ptr(_env.Location()),
			GlobalResourceGroupName:     to.Ptr(azureUniquePrefix + "-global"),
			GlobalSubscriptionID:        to.Ptr(_env.SubscriptionID()),
			MDMFrontendURL:              to.Ptr("https://global.ppe.microsoftmetrics.com/"),
			MDSDEnvironment:             to.Ptr(version.DevGenevaLoggingEnvironment),
			MsiRpEndpoint:               to.Ptr("https://iamaplaceholder.com"),
			MiseValidAudiences: []string{
				"https://management.core.windows.net/",
				_env.Environment().ResourceManagerEndpoint,
			},
			// Azure AD IDs for Apps authorised to send request for authentication via MISE
			MiseValidAppIDs: []string{
				"2187cde1-7e28-4645-9104-19edfa500053",
			},
			PortalAccessGroupIDs: []string{
				os.Getenv("AZURE_PORTAL_ACCESS_GROUP_IDS"),
			},
			PortalClientID: to.Ptr(os.Getenv("AZURE_PORTAL_CLIENT_ID")),
			PortalElevatedGroupIDs: []string{
				os.Getenv("AZURE_PORTAL_ELEVATED_GROUP_IDS"),
			},
			AzureSecPackQualysUrl: to.Ptr(""),
			RPFeatures: []string{
				"DisableDenyAssignments",
				"DisableSignedCertificates",
				"EnableDevelopmentAuthorizer",
				"RequireD2sWorkers",
				"DisableReadinessDelay",
				"RequireOIDCStorageWebEndpoint",
				"UseMockMsiRp",
			},
			// TODO update this to support FF
			RPImagePrefix:                     to.Ptr(azureUniquePrefix + "aro.azurecr.io/aro"),
			RPMDMAccount:                      to.Ptr(version.DevRPGenevaMetricsAccount),
			RPMDSDAccount:                     to.Ptr(version.DevRPGenevaLoggingAccount),
			RPMDSDConfigVersion:               to.Ptr(version.DevRPGenevaLoggingConfigVersion),
			RPMDSDNamespace:                   to.Ptr(version.DevRPGenevaLoggingNamespace),
			RPParentDomainName:                to.Ptr(azureUniquePrefix + "-rp." + os.Getenv("PARENT_DOMAIN_NAME")),
			RPVersionStorageAccountName:       to.Ptr(azureUniquePrefix + "rpversion"),
			RPVMSSCapacity:                    to.Ptr(1),
			SSHPublicKey:                      to.Ptr(string(sshPublicKey)),
			SubscriptionResourceGroupLocation: to.Ptr(_env.Location()),
			SubscriptionResourceGroupName:     to.Ptr(azureUniquePrefix + "-subscription"),
			VMSSCleanupEnabled:                to.Ptr(true),
			VMSize:                            to.Ptr("Standard_D2s_v3"),

			// TODO: Replace with Live Service Configuration in KeyVault
			InstallViaHive:           to.Ptr(os.Getenv("ARO_INSTALL_VIA_HIVE")),
			DefaultInstallerPullspec: to.Ptr(os.Getenv("ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC")),
			AdoptByHive:              to.Ptr(os.Getenv("ARO_ADOPT_BY_HIVE")),
		},
	}, nil
}
