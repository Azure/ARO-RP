package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"encoding/pem"
	"os"

	"k8s.io/utils/ptr"

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
					DatabaseAccountName:         ptr.To(azureUniquePrefix + "-aro-" + _env.Location()),
					GatewayStorageAccountDomain: ptr.To(azureUniquePrefix + "gwy" + _env.Location() + ".blob." + _env.Environment().StorageEndpointSuffix),
					KeyvaultDNSSuffix:           &_env.Environment().KeyVaultDNSSuffix,
					KeyvaultPrefix:              &keyvaultPrefix,
					StorageAccountDomain:        ptr.To(azureUniquePrefix + "aro" + _env.Location() + ".blob." + _env.Environment().StorageEndpointSuffix),
					OIDCStorageAccountName:      ptr.To(oidcStorageAccountName),
				},
			},
		},
		Configuration: &Configuration{
			ACRResourceID:                ptr.To("/subscriptions/" + _env.SubscriptionID() + "/resourceGroups/" + azureUniquePrefix + "-global/providers/Microsoft.ContainerRegistry/registries/" + azureUniquePrefix + "aro"),
			AdminAPICABundle:             ptr.To(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca}))),
			AdminAPIClientCertCommonName: &clientCert.Subject.CommonName,
			ARMAPICABundle:               ptr.To(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca}))),
			ARMAPIClientCertCommonName:   &clientCert.Subject.CommonName,
			ARMClientID:                  ptr.To(os.Getenv("AZURE_ARM_CLIENT_ID")),
			AzureSecPackVSATenantId:      ptr.To(""),
			ClusterMDMAccount:            ptr.To(version.DevClusterGenevaMetricsAccount),
			ClusterMDSDAccount:           ptr.To(version.DevClusterGenevaLoggingAccount),
			ClusterMDSDConfigVersion:     ptr.To(version.DevClusterGenevaLoggingConfigVersion),
			ClusterMDSDNamespace:         ptr.To(version.DevClusterGenevaLoggingNamespace),
			ClusterParentDomainName:      ptr.To(azureUniquePrefix + "-clusters." + os.Getenv("PARENT_DOMAIN_NAME")),
			CosmosDB: &CosmosDBConfiguration{
				StandardProvisionedThroughput: 1000,
				PortalProvisionedThroughput:   400,
				GatewayProvisionedThroughput:  400,
			},
			DisableCosmosDBFirewall: ptr.To(true),
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
			FluentbitImage:       ptr.To(version.FluentbitImage(azureUniquePrefix + "aro." + _env.Environment().ContainerRegistryDNSSuffix)),
			FPClientID:           ptr.To(os.Getenv("AZURE_FP_CLIENT_ID")),
			FPTENANTID:           ptr.To(os.Getenv("AZURE_TENANT_ID")),
			FPServicePrincipalID: ptr.To(os.Getenv("AZURE_FP_SERVICE_PRINCIPAL_ID")),
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
			GatewayMDSDConfigVersion:    ptr.To(version.DevGatewayGenevaLoggingConfigVersion),
			GatewayVMSSCapacity:         ptr.To(1),
			GlobalResourceGroupLocation: ptr.To(_env.Location()),
			GlobalResourceGroupName:     ptr.To(azureUniquePrefix + "-global"),
			GlobalSubscriptionID:        ptr.To(_env.SubscriptionID()),
			MDMFrontendURL:              ptr.To("https://global.ppe.microsoftmetrics.com/"),
			MDSDEnvironment:             ptr.To(version.DevGenevaLoggingEnvironment),
			MsiRpEndpoint:               ptr.To("https://iamaplaceholder.com"),
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
			PortalClientID: ptr.To(os.Getenv("AZURE_PORTAL_CLIENT_ID")),
			PortalElevatedGroupIDs: []string{
				os.Getenv("AZURE_PORTAL_ELEVATED_GROUP_IDS"),
			},
			AzureSecPackQualysUrl: ptr.To(""),
			RPFeatures: []string{
				"DisableDenyAssignments",
				"DisableSignedCertificates",
				"EnableDevelopmentAuthorizer",
				"RequireD2sV3Workers",
				"DisableReadinessDelay",
				"EnableOCMEndpoints",
				"RequireOIDCStorageWebEndpoint",
				"UseMockMsiRp",
				"EnableMISE",
			},
			// TODO update this to support FF
			RPImagePrefix:                     ptr.To(azureUniquePrefix + "aro.azurecr.io/aro"),
			RPMDMAccount:                      ptr.To(version.DevRPGenevaMetricsAccount),
			RPMDSDAccount:                     ptr.To(version.DevRPGenevaLoggingAccount),
			RPMDSDConfigVersion:               ptr.To(version.DevRPGenevaLoggingConfigVersion),
			RPMDSDNamespace:                   ptr.To(version.DevRPGenevaLoggingNamespace),
			RPParentDomainName:                ptr.To(azureUniquePrefix + "-rp." + os.Getenv("PARENT_DOMAIN_NAME")),
			RPVersionStorageAccountName:       ptr.To(azureUniquePrefix + "rpversion"),
			RPVMSSCapacity:                    ptr.To(1),
			SSHPublicKey:                      ptr.To(string(sshPublicKey)),
			SubscriptionResourceGroupLocation: ptr.To(_env.Location()),
			SubscriptionResourceGroupName:     ptr.To(azureUniquePrefix + "-subscription"),
			VMSSCleanupEnabled:                ptr.To(true),
			VMSize:                            ptr.To("Standard_D2s_v3"),

			// TODO: Replace with Live Service Configuration in KeyVault
			InstallViaHive:           ptr.To(os.Getenv("ARO_INSTALL_VIA_HIVE")),
			DefaultInstallerPullspec: ptr.To(os.Getenv("ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC")),
			AdoptByHive:              ptr.To(os.Getenv("ARO_ADOPT_BY_HIVE")),
		},
	}, nil
}
