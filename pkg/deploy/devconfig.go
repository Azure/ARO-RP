package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"encoding/pem"
	"os"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/version"
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
					AzureCloudName:              &_env.Environment().ActualCloudName,
					DatabaseAccountName:         to.StringPtr(azureUniquePrefix + "-aro-" + _env.Location()),
					GatewayStorageAccountDomain: to.StringPtr(azureUniquePrefix + "gwy" + _env.Location() + ".blob." + _env.Environment().StorageEndpointSuffix),
					KeyvaultDNSSuffix:           &_env.Environment().KeyVaultDNSSuffix,
					KeyvaultPrefix:              &keyvaultPrefix,
					StorageAccountDomain:        to.StringPtr(azureUniquePrefix + "aro" + _env.Location() + ".blob." + _env.Environment().StorageEndpointSuffix),
					OIDCStorageAccountName:      to.StringPtr(oidcStorageAccountName),
				},
			},
		},
		Configuration: &Configuration{
			ACRResourceID:                to.StringPtr("/subscriptions/" + _env.SubscriptionID() + "/resourceGroups/" + azureUniquePrefix + "-global/providers/Microsoft.ContainerRegistry/registries/" + azureUniquePrefix + "aro"),
			AdminAPICABundle:             to.StringPtr(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca}))),
			AdminAPIClientCertCommonName: &clientCert.Subject.CommonName,
			ARMAPICABundle:               to.StringPtr(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca}))),
			ARMAPIClientCertCommonName:   &clientCert.Subject.CommonName,
			ARMClientID:                  to.StringPtr(os.Getenv("AZURE_ARM_CLIENT_ID")),
			AzureSecPackVSATenantId:      to.StringPtr(""),
			ClusterMDMAccount:            to.StringPtr(version.DevClusterGenevaMetricsAccount),
			ClusterMDSDAccount:           to.StringPtr(version.DevClusterGenevaLoggingAccount),
			ClusterMDSDConfigVersion:     to.StringPtr(version.DevClusterGenevaLoggingConfigVersion),
			ClusterMDSDNamespace:         to.StringPtr(version.DevClusterGenevaLoggingNamespace),
			ClusterParentDomainName:      to.StringPtr(azureUniquePrefix + "-clusters." + os.Getenv("PARENT_DOMAIN_NAME")),
			CosmosDB: &CosmosDBConfiguration{
				StandardProvisionedThroughput: 1000,
				PortalProvisionedThroughput:   400,
				GatewayProvisionedThroughput:  400,
			},
			DisableCosmosDBFirewall: to.BoolPtr(true),
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
			FluentbitImage:       to.StringPtr(version.FluentbitImage(azureUniquePrefix + "aro." + _env.Environment().ContainerRegistryDNSSuffix)),
			FPClientID:           to.StringPtr(os.Getenv("AZURE_FP_CLIENT_ID")),
			FPTENANTID:           to.StringPtr(os.Getenv("AZURE_TENANT_ID")),
			FPServicePrincipalID: to.StringPtr(os.Getenv("AZURE_FP_SERVICE_PRINCIPAL_ID")),
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
			GatewayMDSDConfigVersion:    to.StringPtr(version.DevGatewayGenevaLoggingConfigVersion),
			GatewayVMSSCapacity:         to.IntPtr(1),
			GlobalResourceGroupLocation: to.StringPtr(_env.Location()),
			GlobalResourceGroupName:     to.StringPtr(azureUniquePrefix + "-global"),
			GlobalSubscriptionID:        to.StringPtr(_env.SubscriptionID()),
			MDMFrontendURL:              to.StringPtr("https://global.ppe.microsoftmetrics.com/"),
			MDSDEnvironment:             to.StringPtr(version.DevGenevaLoggingEnvironment),
			MISEVALIDAUDIENCES: []string{
				"https://management.core.windows.net/",
				_env.Environment().ResourceManagerEndpoint,
			},
			MISEVALIDAPPIDs: []string{
				"2187cde1-7e28-4645-9104-19edfa500053",
			},
			PortalAccessGroupIDs: []string{
				os.Getenv("AZURE_PORTAL_ACCESS_GROUP_IDS"),
			},
			PortalClientID: to.StringPtr(os.Getenv("AZURE_PORTAL_CLIENT_ID")),
			PortalElevatedGroupIDs: []string{
				os.Getenv("AZURE_PORTAL_ELEVATED_GROUP_IDS"),
			},
			AzureSecPackQualysUrl: to.StringPtr(""),
			RPFeatures: []string{
				"DisableDenyAssignments",
				"DisableSignedCertificates",
				"EnableDevelopmentAuthorizer",
				"RequireD2sV3Workers",
				"DisableReadinessDelay",
				"EnableOCMEndpoints",
				"RequireOIDCStorageWebEndpoint",
				"EnableMISE",
			},
			// TODO update this to support FF
			RPImagePrefix:                     to.StringPtr(azureUniquePrefix + "aro.azurecr.io/aro"),
			RPMDMAccount:                      to.StringPtr(version.DevRPGenevaMetricsAccount),
			RPMDSDAccount:                     to.StringPtr(version.DevRPGenevaLoggingAccount),
			RPMDSDConfigVersion:               to.StringPtr(version.DevRPGenevaLoggingConfigVersion),
			RPMDSDNamespace:                   to.StringPtr(version.DevRPGenevaLoggingNamespace),
			RPParentDomainName:                to.StringPtr(azureUniquePrefix + "-rp." + os.Getenv("PARENT_DOMAIN_NAME")),
			RPVersionStorageAccountName:       to.StringPtr(azureUniquePrefix + "rpversion"),
			RPVMSSCapacity:                    to.IntPtr(1),
			SSHPublicKey:                      to.StringPtr(string(sshPublicKey)),
			SubscriptionResourceGroupLocation: to.StringPtr(_env.Location()),
			SubscriptionResourceGroupName:     to.StringPtr(azureUniquePrefix + "-subscription"),
			VMSSCleanupEnabled:                to.BoolPtr(true),
			VMSize:                            to.StringPtr("Standard_D2s_v3"),

			// TODO: Replace with Live Service Configuration in KeyVault
			InstallViaHive:           to.StringPtr(os.Getenv("ARO_INSTALL_VIA_HIVE")),
			DefaultInstallerPullspec: to.StringPtr(os.Getenv("ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC")),
			AdoptByHive:              to.StringPtr(os.Getenv("ARO_ADOPT_BY_HIVE")),
		},
	}, nil
}
