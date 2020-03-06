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
	RPS           []RPConfig    `json:"rps"`
	Configuration Configuration `json:"configuration"`
}

// RPConfig represents individual RP configuration
type RPConfig struct {
	// Name is unique identifier of RP
	Name              string        `json:"name"`
	Location          string        `json:"location"`
	ResourceGroupName string        `json:"resourceGroupName"`
	SubscriptionID    string        `json:"subscriptionId"`
	Configuration     Configuration `json:"configuration"`
}

// Configuration represents configuration structure
type Configuration struct {
	AdminAPIClientCertCommonName string        `json:"adminApiClientCertCommonName"`
	AdminAPICaBundle             string        `json:"adminApiCaBundle"`
	KeyvaultPrefix               string        `json:"keyvaultPrefix"`
	RPServicePrincipalID         string        `json:"rpServicePrincipalId"`
	FPServicePrincipalID         string        `json:"fpServicePrincipalId"`
	ExtraKeyvaultAccessPolicies  []interface{} `json:"extraKeyvaultAccessPolicies"`
	RpImage                      string        `json:"rpImage"`
	RpMode                       string        `json:"rpMode"`
	SSHPublicKey                 string        `json:"sshPublicKey"`
	VMSSName                     string        `json:"vmssName"`
	MDMFrontendURL               string        `json:"mdmFrontendUrl"`
	DatabaseAccountName          string        `json:"databaseAccountName"`
	ExtraCosmosDBIPs             string        `json:"extraCosmosDBIPs"`
	PullSecret                   string        `json:"pullSecret"`
	DomainName                   string        `json:"domainName"`
	MDSDConfigVersion            string        `json:"mdsdConfigVersion"`
	RPImageAuth                  string        `json:"rpImageAuth"`
	MDSDEnvironment              string        `json:"mdsdEnvironment"`
}

// GetConfig return RP configuration from the file
func GetConfig(region, path string) (*RPConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config *Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	for _, c := range config.RPS {
		if c.Name == region {
			configuration, err := mergeConfig(&c.Configuration, &config.Configuration)
			if err != nil {
				return nil, err
			}
			c.Configuration = *configuration
			return &c, nil
		}
	}

	return nil, fmt.Errorf("region %s not found in the %s", region, path)
}

// mergeConfig merges two Configuration structs, where Primary input
// takes priority over secondary
func mergeConfig(primary *Configuration, secondary *Configuration) (*Configuration, error) {
	if reflect.ValueOf(primary).IsNil() || reflect.ValueOf(secondary).IsNil() {
		return nil, fmt.Errorf("inputs can't be nil")
	}
	sValues := reflect.Indirect(reflect.ValueOf(secondary))
	pValues := reflect.Indirect(reflect.ValueOf(primary))

	for i := 0; i < pValues.NumField(); i++ {
		if pValues.Field(i).IsZero() {
			pValues.Field(i).Set(sValues.Field(i))
		}
	}

	return primary, nil
}
