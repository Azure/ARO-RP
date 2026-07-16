package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"encoding/pem"
	"os"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

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
					DatabaseAccountName:    new(azureUniquePrefix + "-aro-" + _env.Location()),
					KeyvaultDNSSuffix:      &_env.Environment().KeyVaultDNSSuffix,
					KeyvaultPrefix:         &keyvaultPrefix,
					OIDCStorageAccountName: new(oidcStorageAccountName),
					OtelAuditQueueSize:     new("0"),
				},
			},
		},
		Configuration: &Configuration{
			ACRResourceID:                new("/subscriptions/" + _env.SubscriptionID() + "/resourceGroups/" + azureUniquePrefix + "-global/providers/Microsoft.ContainerRegistry/registries/" + azureUniquePrefix + "aro"),
			AdminAPICABundle:             new(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca}))),
			AdminAPIClientCertCommonName: &clientCert.Subject.CommonName,
			ARMAPICABundle:               new(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca}))),
			ARMAPIClientCertCommonName:   &clientCert.Subject.CommonName,
			ARMClientID:                  new(os.Getenv("AZURE_ARM_CLIENT_ID")),
			AzureSecPackVSATenantId:      new(""),
			ClusterMDMAccount:            pointerutils.ToPtr(version.DevClusterGenevaMetricsAccount),
			ClusterMDSDAccount:           pointerutils.ToPtr(version.DevClusterGenevaLoggingAccount),
			ClusterMDSDConfigVersion:     pointerutils.ToPtr(version.DevClusterGenevaLoggingConfigVersion),
			ClusterMDSDNamespace:         pointerutils.ToPtr(version.DevClusterGenevaLoggingNamespace),
			ClusterParentDomainName:      new(azureUniquePrefix + "-clusters." + os.Getenv("PARENT_DOMAIN_NAME")),
			CosmosDB: &CosmosDBConfiguration{
				StandardProvisionedThroughput: 1000,
				PortalProvisionedThroughput:   400,
				GatewayProvisionedThroughput:  400,
			},
			DisableCosmosDBFirewall: new(true),
			FluentbitImage:          new(version.FluentbitImage(azureUniquePrefix + "aro." + _env.Environment().ContainerRegistryDNSSuffix)),
			FPClientID:              new(os.Getenv("AZURE_FP_CLIENT_ID")),
			FPServicePrincipalID:    new(os.Getenv("AZURE_FP_SERVICE_PRINCIPAL_ID")),
			FPTenantID:              new(os.Getenv("AZURE_TENANT_ID")),
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
			GatewayMDSDConfigVersion:    pointerutils.ToPtr(version.DevGatewayGenevaLoggingConfigVersion),
			GatewayVMSSCapacity:         new(1),
			GlobalResourceGroupLocation: new(_env.Location()),
			GlobalResourceGroupName:     new(azureUniquePrefix + "-global"),
			GlobalSubscriptionID:        new(_env.SubscriptionID()),
			MDMFrontendURL:              new("https://global.ppe.microsoftmetrics.com/"),
			MDSDEnvironment:             pointerutils.ToPtr(version.DevGenevaLoggingEnvironment),
			MsiRpEndpoint:               new("https://iamaplaceholder.com"),
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
			PortalClientID: new(os.Getenv("AZURE_PORTAL_CLIENT_ID")),
			PortalElevatedGroupIDs: []string{
				os.Getenv("AZURE_PORTAL_ELEVATED_GROUP_IDS"),
			},
			AzureSecPackQualysUrl: new(""),
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
			RPImagePrefix:                     new(azureUniquePrefix + "aro.azurecr.io/aro"),
			RPMDMAccount:                      pointerutils.ToPtr(version.DevRPGenevaMetricsAccount),
			RPMDSDAccount:                     pointerutils.ToPtr(version.DevRPGenevaLoggingAccount),
			RPMDSDConfigVersion:               pointerutils.ToPtr(version.DevRPGenevaLoggingConfigVersion),
			RPMDSDNamespace:                   pointerutils.ToPtr(version.DevRPGenevaLoggingNamespace),
			RPParentDomainName:                new(azureUniquePrefix + "-rp." + os.Getenv("PARENT_DOMAIN_NAME")),
			RPVersionStorageAccountName:       new(azureUniquePrefix + "rpversion"),
			RPVMSSCapacity:                    new(1),
			SSHPublicKey:                      new(string(sshPublicKey)),
			SubscriptionResourceGroupLocation: new(_env.Location()),
			SubscriptionResourceGroupName:     new(azureUniquePrefix + "-subscription"),
			VMSSCleanupEnabled:                new(true),
			TokenContributorRoleID:            new("48983534-3d06-4dcb-a566-08a694eb1279"),
			TokenContributorRoleName:          new("ARO v4 ContainerRegistry Token Contributor"),
			VMSize:                            new("Standard_D2s_v3"),

			// TODO: Replace with Live Service Configuration in KeyVault
			InstallViaHive:           new(os.Getenv("ARO_INSTALL_VIA_HIVE")),
			DefaultInstallerPullspec: new(os.Getenv("ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC")),
			AdoptByHive:              new(os.Getenv("ARO_ADOPT_BY_HIVE")),
		},
	}, nil
}
