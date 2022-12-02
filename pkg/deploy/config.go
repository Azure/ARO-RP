package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	"golang.org/x/crypto/ssh"
)

// NOTICE: when modifying the config definition here, don't forget to update
// DevConfig().

// Config represents configuration object for deployer tooling
type Config struct {
	RPs           []RPConfig     `json:"rps,omitempty"`
	Configuration *Configuration `json:"configuration,omitempty"`
}

// RPConfig represents individual RP configuration
type RPConfig struct {
	Location                 string         `json:"location,omitempty"`
	SubscriptionID           string         `json:"subscriptionId,omitempty"`
	GatewayResourceGroupName string         `json:"gatewayResourceGroupName,omitempty"`
	RPResourceGroupName      string         `json:"rpResourceGroupName,omitempty"`
	Configuration            *Configuration `json:"configuration,omitempty"`
}

// Configuration represents configuration structure
type Configuration struct {
	ACRLocationOverride                *string       `json:"acrLocationOverride,omitempty"`
	ACRResourceID                      *string       `json:"acrResourceId,omitempty" value:"required"`
	AzureCloudName                     *string       `json:"azureCloudName,omitempty" value:"required"`
	AzureSecPackQualysUrl              *string       `json:"azureSecPackQualysUrl,omitempty"`
	AzureSecPackVSATenantId            *string       `json:"azureSecPackVSATenantId,omitempty"`
	RPVersionStorageAccountName        *string       `json:"rpVersionStorageAccountName,omitempty" value:"required"`
	ACRReplicaDisabled                 *bool         `json:"acrReplicaDisabled,omitempty"`
	AdminAPICABundle                   *string       `json:"adminApiCaBundle,omitempty" value:"required"`
	AdminAPIClientCertCommonName       *string       `json:"adminApiClientCertCommonName,omitempty" value:"required"`
	ARMAPICABundle                     *string       `json:"armApiCaBundle,omitempty"`
	ARMAPIClientCertCommonName         *string       `json:"armApiClientCertCommonName,omitempty"`
	ARMClientID                        *string       `json:"armClientId,omitempty"`
	BillingE2EStorageAccountID         *string       `json:"billingE2EStorageAccountId,omitempty"`
	BillingServicePrincipalID          *string       `json:"billingServicePrincipalId,omitempty"`
	ClusterMDMAccount                  *string       `json:"clusterMdmAccount,omitempty" value:"required"`
	ClusterMDSDAccount                 *string       `json:"clusterMdsdAccount,omitempty" value:"required"`
	ClusterMDSDConfigVersion           *string       `json:"clusterMdsdConfigVersion,omitempty" value:"required"`
	ClusterMDSDNamespace               *string       `json:"clusterMdsdNamespace,omitempty" value:"required"`
	ClusterParentDomainName            *string       `json:"clusterParentDomainName,omitempty" value:"required"`
	DatabaseAccountName                *string       `json:"databaseAccountName,omitempty" value:"required"`
	DBTokenClientID                    *string       `json:"dbtokenClientId,omitempty" value:"required"`
	DisableCosmosDBFirewall            *bool         `json:"disableCosmosDBFirewall,omitempty"`
	ExtraClusterKeyvaultAccessPolicies []interface{} `json:"extraClusterKeyvaultAccessPolicies,omitempty" value:"required"`
	ExtraDBTokenKeyvaultAccessPolicies []interface{} `json:"extraDBTokenKeyvaultAccessPolicies,omitempty" value:"required"`
	ExtraCosmosDBIPs                   []string      `json:"extraCosmosDBIPs,omitempty"`
	ExtraGatewayKeyvaultAccessPolicies []interface{} `json:"extraGatewayKeyvaultAccessPolicies,omitempty" value:"required"`
	ExtraPortalKeyvaultAccessPolicies  []interface{} `json:"extraPortalKeyvaultAccessPolicies,omitempty" value:"required"`
	ExtraServiceKeyvaultAccessPolicies []interface{} `json:"extraServiceKeyvaultAccessPolicies,omitempty" value:"required"`
	FluentbitImage                     *string       `json:"fluentbitImage,omitempty" value:"required"`
	FPClientID                         *string       `json:"fpClientId,omitempty" value:"required"`
	FPServerCertCommonName             *string       `json:"fpServerCertCommonName,omitempty"`
	FPServicePrincipalID               *string       `json:"fpServicePrincipalId,omitempty" value:"required"`
	GatewayDomains                     []string      `json:"gatewayDomains,omitempty"`
	GatewayFeatures                    []string      `json:"gatewayFeatures,omitempty"`
	GatewayMDSDConfigVersion           *string       `json:"gatewayMdsdConfigVersion,omitempty" value:"required"`
	GatewayStorageAccountDomain        *string       `json:"gatewayStorageAccountDomain,omitempty" value:"required"`
	GatewayVMSize                      *string       `json:"gatewayVmSize,omitempty"`
	GatewayVMSSCapacity                *int          `json:"gatewayVmssCapacity,omitempty"`
	GlobalResourceGroupName            *string       `json:"globalResourceGroupName,omitempty" value:"required"`
	GlobalResourceGroupLocation        *string       `json:"globalResourceGroupLocation,omitempty" value:"required"`
	GlobalSubscriptionID               *string       `json:"globalSubscriptionId,omitempty" value:"required"`
	KeyvaultDNSSuffix                  *string       `json:"keyvaultDNSSuffix,omitempty" value:"required"`
	KeyvaultPrefix                     *string       `json:"keyvaultPrefix,omitempty" value:"required"`
	MDMFrontendURL                     *string       `json:"mdmFrontendUrl,omitempty" value:"required"`
	MDSDEnvironment                    *string       `json:"mdsdEnvironment,omitempty" value:"required"`
	NonZonalRegions                    []string      `json:"nonZonalRegions,omitempty"`
	PortalAccessGroupIDs               []string      `json:"portalAccessGroupIds,omitempty" value:"required"`
	PortalClientID                     *string       `json:"portalClientId,omitempty" value:"required"`
	PortalElevatedGroupIDs             []string      `json:"portalElevatedGroupIds,omitempty" value:"required"`
	RPFeatures                         []string      `json:"rpFeatures,omitempty"`
	RPImagePrefix                      *string       `json:"rpImagePrefix,omitempty" value:"required"`
	RPMDMAccount                       *string       `json:"rpMdmAccount,omitempty" value:"required"`
	RPMDSDAccount                      *string       `json:"rpMdsdAccount,omitempty" value:"required"`
	RPMDSDConfigVersion                *string       `json:"rpMdsdConfigVersion,omitempty" value:"required"`
	RPMDSDNamespace                    *string       `json:"rpMdsdNamespace,omitempty" value:"required"`
	RPNSGPortalSourceAddressPrefixes   []string      `json:"rpNsgPortalSourceAddressPrefixes,omitempty"`
	RPParentDomainName                 *string       `json:"rpParentDomainName,omitempty" value:"required"`
	RPVMSSCapacity                     *int          `json:"rpVmssCapacity,omitempty"`
	SSHPublicKey                       *string       `json:"sshPublicKey,omitempty"`
	StorageAccountDomain               *string       `json:"storageAccountDomain,omitempty" value:"required"`
	SubscriptionResourceGroupName      *string       `json:"subscriptionResourceGroupName,omitempty" value:"required"`
	SubscriptionResourceGroupLocation  *string       `json:"subscriptionResourceGroupLocation,omitempty" value:"required"`
	VMSize                             *string       `json:"vmSize,omitempty" value:"required"`
	VMSSCleanupEnabled                 *bool         `json:"vmssCleanupEnabled,omitempty"`

	// TODO: Replace with Live Service Configuration in KeyVault
	InstallViaHive           *string `json:"clustersInstallViaHive,omitempty"`
	DefaultInstallerPullspec *string `json:"clusterDefaultInstallerPullspec,omitempty"`
	AdoptByHive              *string `json:"clustersAdoptByHive,omitempty"`
}

// GetConfig return RP configuration from the file
func GetConfig(path, location string) (*RPConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config *Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	for _, c := range config.RPs {
		if c.Location == location {
			configuration, err := mergeConfig(c.Configuration, config.Configuration)
			if err != nil {
				return nil, err
			}

			c.Configuration = configuration
			return &c, nil
		}
	}

	return nil, fmt.Errorf("location %s not found in %s", location, path)
}

// mergeConfig merges two Configuration structs, replacing each zero field in
// primary with the contents of the corresponding field in secondary
func mergeConfig(primary, secondary *Configuration) (*Configuration, error) {
	sValues := reflect.ValueOf(secondary).Elem()
	pValues := reflect.ValueOf(primary).Elem()

	for i := 0; i < pValues.NumField(); i++ {
		if pValues.Field(i).IsZero() {
			pValues.Field(i).Set(sValues.Field(i))
		}
	}

	return primary, nil
}

// CheckRequiredFields validates configuration whether it provides required fields.
// Config is invalid if required fields are not provided.
func (conf *RPConfig) validate() error {
	configuration := conf.Configuration
	v := reflect.ValueOf(*configuration)
	missingFields := []string{}

	if conf.Configuration.SSHPublicKey == nil {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return err
		}
		publicRsaKey, err := ssh.NewPublicKey(&key.PublicKey)
		if err != nil {
			return err
		}
		publicKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)
		conf.Configuration.SSHPublicKey = to.StringPtr(string(publicKeyBytes))
	}

	for i := 0; i < v.NumField(); i++ {
		required := v.Type().Field(i).Tag.Get("value") == "required"

		if required && v.Field(i).IsZero() {
			missingFields = append(missingFields, v.Type().Field(i).Name)
		}
	}

	if len(missingFields) == 0 {
		return nil
	}

	return fmt.Errorf("configuration has missing fields: %s", strings.Join(missingFields, ","))
}
