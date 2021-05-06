package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
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
				"List",
				"Set",
			},
		},
		"tenantId": _env.TenantID(),
	}
}

func DevConfig(_env env.Core) (*Config, error) {
	ca, err := ioutil.ReadFile("secrets/dev-ca.crt")
	if err != nil {
		return nil, err
	}

	client, err := ioutil.ReadFile("secrets/dev-client.crt")
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

	sshPublicKey, err := ioutil.ReadFile(sshPublicKeyPath)
	if err != nil {
		return nil, err
	}

	keyvaultPrefix := os.Getenv("USER") + "-aro-" + _env.Location()
	if len(keyvaultPrefix) > 20 {
		keyvaultPrefix = keyvaultPrefix[:20]
	}

	return &Config{
		RPs: []RPConfig{
			{
				Location:            _env.Location(),
				SubscriptionID:      _env.SubscriptionID(),
				RPResourceGroupName: os.Getenv("USER") + "-aro-" + _env.Location(),
				Configuration: &Configuration{
					DatabaseAccountName:  to.StringPtr(os.Getenv("USER") + "-aro-" + _env.Location()),
					KeyvaultPrefix:       &keyvaultPrefix,
					StorageAccountDomain: to.StringPtr(os.Getenv("USER") + "aro" + _env.Location() + ".blob." + _env.Environment().StorageEndpointSuffix),
				},
			},
		},
		Configuration: &Configuration{
			ACRResourceID:                to.StringPtr("/subscriptions/" + _env.SubscriptionID() + "/resourceGroups/" + os.Getenv("USER") + "-global/providers/Microsoft.ContainerRegistry/registries/" + os.Getenv("USER") + "aro"),
			AdminAPICABundle:             to.StringPtr(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca}))),
			AdminAPIClientCertCommonName: to.StringPtr(clientCert.Subject.CommonName),
			ARMAPICABundle:               to.StringPtr(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca}))),
			ARMAPIClientCertCommonName:   to.StringPtr(clientCert.Subject.CommonName),
			ARMClientID:                  to.StringPtr(os.Getenv("AZURE_ARM_CLIENT_ID")),
			ClusterMDSDConfigVersion:     to.StringPtr(version.DevClusterGenevaLoggingConfigVersion),
			ClusterParentDomainName:      to.StringPtr(os.Getenv("USER") + "-clusters." + os.Getenv("PARENT_DOMAIN_NAME")),
			DisableCosmosDBFirewall:      to.BoolPtr(true),
			ExtraClusterKeyvaultAccessPolicies: []interface{}{
				adminKeyvaultAccessPolicy(_env),
			},
			ExtraDBTokenKeyvaultAccessPolicies: []interface{}{
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
			FPClientID:                  to.StringPtr(os.Getenv("AZURE_FP_CLIENT_ID")),
			FPServicePrincipalID:        to.StringPtr(os.Getenv("AZURE_FP_SERVICE_PRINCIPAL_ID")),
			GlobalResourceGroupLocation: to.StringPtr(_env.Location()),
			GlobalResourceGroupName:     to.StringPtr(os.Getenv("USER") + "-global"),
			GlobalSubscriptionID:        to.StringPtr(_env.SubscriptionID()),
			MDMFrontendURL:              to.StringPtr("https://int2.int.microsoftmetrics.com/"),
			MDSDEnvironment:             to.StringPtr(version.DevClusterGenevaLoggingEnvironment),
			PortalAccessGroupIDs: []string{
				os.Getenv("AZURE_PORTAL_ACCESS_GROUP_IDS"),
			},
			PortalClientID: to.StringPtr(os.Getenv("AZURE_PORTAL_CLIENT_ID")),
			PortalElevatedGroupIDs: []string{
				os.Getenv("AZURE_PORTAL_ELEVATED_GROUP_IDS"),
			},
			RPFeatures: []string{
				"DisableDenyAssignments",
				"DisableSignedCertificates",
				"EnableDevelopmentAuthorizer",
				"RequireD2sV3Workers",
				"DisableReadinessDelay",
			},
			RPImagePrefix:       to.StringPtr(os.Getenv("USER") + "aro.azurecr.io/aro"),
			RPMDSDConfigVersion: to.StringPtr("3.3"),
			RPNSGSourceAddressPrefixes: []string{
				"0.0.0.0/0",
			},
			RPParentDomainName:                to.StringPtr(os.Getenv("USER") + "-rp." + os.Getenv("PARENT_DOMAIN_NAME")),
			RPVersionStorageAccountName:       to.StringPtr(os.Getenv("USER") + "rpversion"),
			RPVMSSCapacity:                    to.IntPtr(1),
			SSHPublicKey:                      to.StringPtr(string(sshPublicKey)),
			SubscriptionResourceGroupLocation: to.StringPtr(_env.Location()),
			SubscriptionResourceGroupName:     to.StringPtr(os.Getenv("USER") + "-subscription"),
			VMSize:                            to.StringPtr("Standard_D2s_v3"),
		},
	}, nil
}
