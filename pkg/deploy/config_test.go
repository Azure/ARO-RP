package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func TestConfigurationFieldParity(t *testing.T) {
	// create a map whose keys are all the fields of Configuration
	m := map[string]struct{}{}

	typ := reflect.TypeOf(Configuration{})
	for i := 0; i < typ.NumField(); i++ {
		m[strings.SplitN(typ.Field(i).Tag.Get("json"), ",", 2)[0]] = struct{}{}
	}

	for _, paramsFile := range []string{
		generator.FileRPProductionParameters,
		generator.FileRPProductionPredeployParameters,
	} {
		b, err := Asset(paramsFile)
		if err != nil {
			t.Fatal(err)
		}

		var params *arm.Parameters
		err = json.Unmarshal(b, &params)
		if err != nil {
			t.Fatal(err)
		}

		// check each parameter exists as a field in Configuration
		for name := range params.Parameters {
			switch name {
			case "deployNSGs", "domainName", "fullDeploy", "rpImage", "rpServicePrincipalId", "vmssName":
			default:
				if _, found := m[name]; !found {
					t.Errorf("field %s not found in config.Configuration but exists in templates", name)
				}
			}
		}
	}
}

func TestMergeConfig(t *testing.T) {
	databaseAccountName := to.StringPtr("databaseAccountName")
	fpServerCertCommonName := to.StringPtr("fpServerCertCommonName")
	fpServerSecondaryCommonName := to.StringPtr("fpServerSecondaryCommonName")
	kvPrefix := to.StringPtr("keyvaultPrefix")

	for _, tt := range []struct {
		name      string
		primary   Configuration
		secondary Configuration
		want      Configuration
	}{
		{
			name: "noop",
		},
		{
			name: "overrides",
			primary: Configuration{
				DatabaseAccountName:    databaseAccountName,
				FPServerCertCommonName: fpServerCertCommonName,
			},
			secondary: Configuration{
				FPServerCertCommonName: fpServerSecondaryCommonName,
				KeyvaultPrefix:         kvPrefix,
			},
			want: Configuration{
				DatabaseAccountName:    databaseAccountName,
				FPServerCertCommonName: fpServerCertCommonName,
				KeyvaultPrefix:         kvPrefix,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeConfig(&tt.primary, &tt.secondary)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(&tt.want, got) {
				t.Fatalf("%#v", got)
			}
		})
	}
}

func TestConfigNilable(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Configuration can contain only nilable types. %v", r)
		}
	}()

	cfg := Configuration{}
	val := reflect.ValueOf(cfg)

	for i := 0; i < val.NumField(); i++ {
		val.Field(i).IsNil()
	}
}

func TestConfigRequiredValues(t *testing.T) {
	AdminAPICABundle := "AdminAPICABundle"
	ACRResourceID := "ACRResourceID"
	ExtraCosmosDBIPs := "ExtraCosmosDBIPs"
	MDMFrontendURL := "MDMFrontendURL"
	AdminAPIClientCertCommonName := "AdminAPIClientCertCommonName"
	ClusterParentDomainName := "ClusterParentDomainName"
	DatabaseAccountName := "DatabaseAccountName"
	FPServerCertCommonName := "FPServerCertCommonName"
	FPServicePrincipalID := "FPServicePrincipalID"
	GlobalMonitoringKeyVaultURI := "GlobalMonitoringKeyVaultURI"
	GlobalResourceGroupName := "GlobalResourceGroupName"
	GlobalSubscriptionID := "GlobalSubscriptionID"
	KeyvaultPrefix := "KeyvaultPrefix"
	MDSDConfigVersion := "MDSDConfigVersion"
	MDSDEnvironment := "MDSDEnvironment"
	RPImagePrefix := "RPImagePrefix"
	RPMode := "RPMode"
	RPParentDomainName := "RPParentDomainName"
	RPVersionStorageAccountName := "RPVersionStorageAccountName"
	SSHPublicKey := "SSHPublicKey"
	SubscriptionResourceGroupName := "SubscriptionResourceGroupName"
	VMSize := "VMSize"

	for _, tt := range []struct {
		name   string
		config RPConfig
		expect error
	}{
		{
			name: "valid config",
			config: RPConfig{
				Configuration: &Configuration{
					ACRResourceID:                      &ACRResourceID,
					AdminAPICABundle:                   &AdminAPICABundle,
					ExtraCosmosDBIPs:                   []string{ExtraCosmosDBIPs},
					MDMFrontendURL:                     &MDMFrontendURL,
					ACRReplicaDisabled:                 to.BoolPtr(true),
					AdminAPIClientCertCommonName:       &AdminAPIClientCertCommonName,
					ClusterParentDomainName:            &ClusterParentDomainName,
					DatabaseAccountName:                &DatabaseAccountName,
					ExtraClusterKeyvaultAccessPolicies: []interface{}{},
					ExtraServiceKeyvaultAccessPolicies: []interface{}{},
					FPServerCertCommonName:             &FPServerCertCommonName,
					FPServicePrincipalID:               &FPServicePrincipalID,
					GlobalMonitoringKeyVaultURI:        &GlobalMonitoringKeyVaultURI,
					GlobalResourceGroupName:            &GlobalResourceGroupName,
					GlobalSubscriptionID:               &GlobalSubscriptionID,
					KeyvaultPrefix:                     &KeyvaultPrefix,
					MDSDConfigVersion:                  &MDSDConfigVersion,
					MDSDEnvironment:                    &MDSDEnvironment,
					RPImagePrefix:                      &RPImagePrefix,
					RPMode:                             &RPMode,
					RPNSGSourceAddressPrefixes:         []string{},
					RPParentDomainName:                 &RPParentDomainName,
					RPVersionStorageAccountName:        &RPVersionStorageAccountName,
					SSHPublicKey:                       &SSHPublicKey,
					SubscriptionResourceGroupName:      &SubscriptionResourceGroupName,
					VMSize:                             &VMSize,
				},
			},
			expect: nil,
		},
		{
			name: "invalid config",
			config: RPConfig{
				Configuration: &Configuration{
					ACRResourceID:    &ACRResourceID,
					AdminAPICABundle: &AdminAPICABundle,
					ExtraCosmosDBIPs: []string{ExtraCosmosDBIPs},
				},
			},
			expect: fmt.Errorf("Configuration has missing fields: %s", "[RPVersionStorageAccountName AdminAPIClientCertCommonName ClusterParentDomainName DatabaseAccountName ExtraClusterKeyvaultAccessPolicies ExtraServiceKeyvaultAccessPolicies FPServicePrincipalID GlobalMonitoringKeyVaultURI GlobalResourceGroupName GlobalSubscriptionID KeyvaultPrefix MDMFrontendURL MDSDConfigVersion MDSDEnvironment RPImagePrefix RPNSGSourceAddressPrefixes RPParentDomainName SubscriptionResourceGroupName SSHPublicKey VMSize]"),
		},
	} {
		valid := tt.config.validate()
		if valid != tt.expect && valid.Error() != tt.expect.Error() {
			t.Errorf("Expected %s but got %s", tt.name, valid.Error())
		}
	}
}
