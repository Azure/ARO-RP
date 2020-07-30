package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"io/ioutil"
	"reflect"

	"github.com/ghodss/yaml"
)

// Config represents configuration object for deployer tooling
type Config struct {
	RPs           []RPConfig     `json:"rps,omitempty"`
	Configuration *Configuration `json:"configuration,omitempty"`
}

// RPConfig represents individual RP configuration
type RPConfig struct {
	Location          string         `json:"location,omitempty"`
	SubscriptionID    string         `json:"subscriptionId,omitempty"`
	ResourceGroupName string         `json:"resourceGroupName,omitempty"`
	Configuration     *Configuration `json:"configuration,omitempty"`
}

// Configuration represents configuration structure
type Configuration struct {
	ACRResourceID                      string        `json:"acrResourceId,omitempty"`
	ACRReplica                         bool          `json:"acrReplica,omitempty"`
	AdminAPICABundle                   string        `json:"adminApiCaBundle,omitempty"`
	AdminAPIClientCertCommonName       string        `json:"adminApiClientCertCommonName,omitempty"`
	ClusterParentDomainName            string        `json:"clusterParentDomainName,omitempty"`
	DatabaseAccountName                string        `json:"databaseAccountName,omitempty"`
	ExtraClusterKeyvaultAccessPolicies []interface{} `json:"extraClusterKeyvaultAccessPolicies,omitempty"`
	ExtraCosmosDBIPs                   []string      `json:"extraCosmosDBIPs,omitempty"`
	ExtraServiceKeyvaultAccessPolicies []interface{} `json:"extraServiceKeyvaultAccessPolicies,omitempty"`
	FPServerCertCommonName             string        `json:"fpServerCertCommonName,omitempty"`
	FPServicePrincipalID               string        `json:"fpServicePrincipalId,omitempty"`
	GlobalMonitoringKeyVaultURI        string        `json:"globalMonitoringKeyVaultUri,omitempty"`
	GlobalResourceGroupName            string        `json:"globalResourceGroupName,omitempty"`
	GlobalSubscriptionID               string        `json:"globalSubscriptionId,omitempty"`
	KeyvaultPrefix                     string        `json:"keyvaultPrefix,omitempty"`
	MDMFrontendURL                     string        `json:"mdmFrontendUrl,omitempty"`
	MDSDConfigVersion                  string        `json:"mdsdConfigVersion,omitempty"`
	MDSDEnvironment                    string        `json:"mdsdEnvironment,omitempty"`
	RPImagePrefix                      string        `json:"rpImagePrefix,omitempty"`
	RPMode                             string        `json:"rpMode,omitempty"`
	RPNSGSourceAddressPrefixes         []string      `json:"rpNsgSourceAddressPrefixes,omitempty"`
	RPParentDomainName                 string        `json:"rpParentDomainName,omitempty"`
	SubscriptionResourceGroupName      string        `json:"subscriptionResourceGroupName,omitempty"`
	SSHPublicKey                       string        `json:"sshPublicKey,omitempty"`
	VMSize                             string        `json:"vmSize,omitempty"`
}

// GetConfig return RP configuration from the file
func GetConfig(path, location string) (*RPConfig, error) {
	data, err := ioutil.ReadFile(path)
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
