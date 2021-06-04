package instancemetadata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"
)

type TestEnv struct {
	data map[string]string
}

func NewTestEnv(envMap map[string]string) TestEnv {
	return TestEnv{
		data: envMap,
	}
}

func (t TestEnv) Getenv(key string) string {
	return t.data[key]
}
func (t TestEnv) LookupEnv(key string) (string, bool) {
	if out, ok := t.data[key]; ok {
		return out, true
	} else {
		return "", false
	}
}

func TestProdEnvPopulateInstanceMetadata(t *testing.T) {
	tests := []struct {
		name        string
		environment map[string]string
		expectErr   bool
	}{
		{
			name:        "no environment",
			environment: make(map[string]string),
			expectErr:   true,
		},
		{
			name: "env missing all keys",
			environment: map[string]string{
				"someKey": "someValue",
			},
			expectErr: true,
		},
		{
			name: "env missing some keys",
			environment: map[string]string{
				"RESOURCEGROUP": "my-rg",
			},
			expectErr: true,
		},
		{
			name: "env public cloud",
			environment: map[string]string{
				"AZURE_ENVIRONMENT":     azure.PublicCloud.Name,
				"AZURE_SUBSCRIPTION_ID": "some-sub-guid",
				"AZURE_TENANT_ID":       "some-tenant-guid",
				"LOCATION":              "some-region",
				"RESOURCEGROUP":         "my-resourceGroup",
			},
			expectErr: false,
		},
		{
			name: "env fairfax",
			environment: map[string]string{
				"AZURE_ENVIRONMENT":     azure.USGovernmentCloud.Name,
				"AZURE_SUBSCRIPTION_ID": "some-sub-guid",
				"AZURE_TENANT_ID":       "some-tenant-guid",
				"LOCATION":              "some-region",
				"RESOURCEGROUP":         "my-resourceGroup",
			},
			expectErr: false,
		},
		{
			name: "env mooncake",
			environment: map[string]string{
				"AZURE_ENVIRONMENT":     azure.ChinaCloud.Name,
				"AZURE_SUBSCRIPTION_ID": "some-sub-guid",
				"AZURE_TENANT_ID":       "some-tenant-guid",
				"LOCATION":              "some-region",
				"RESOURCEGROUP":         "my-resourceGroup",
			},
			expectErr: false,
		},
		{
			name: "env blackforest",
			environment: map[string]string{
				"AZURE_ENVIRONMENT":     azure.GermanCloud.Name,
				"AZURE_SUBSCRIPTION_ID": "some-sub-guid",
				"AZURE_TENANT_ID":       "some-tenant-guid",
				"LOCATION":              "some-region",
				"RESOURCEGROUP":         "my-resourceGroup",
			},
			expectErr: false,
		},
		{
			name: "env with hostname override",
			environment: map[string]string{
				"AZURE_ENVIRONMENT":     azure.PublicCloud.Name,
				"AZURE_SUBSCRIPTION_ID": "some-sub-guid",
				"AZURE_TENANT_ID":       "some-tenant-guid",
				"LOCATION":              "some-region",
				"RESOURCEGROUP":         "my-resourceGroup",
				"HOSTNAME_OVERRIDE":     "my.over.ride",
			},
			expectErr: false,
		},
	}

	for _, test := range tests {
		testEnv := NewTestEnv(test.environment)
		p := &prodenv{}
		err := p.populateInstanceMetadata(testEnv)
		if test.expectErr != (err != nil) {
			t.Errorf("%s: expected error %#v got %#v", test.name, test.expectErr, err)
		} else if !test.expectErr {
			// verify there are values for all required fields
			if p.environment == nil {
				t.Errorf("%s: environment expected, found nil", test.name)
			}
			if p.subscriptionID == "" {
				t.Errorf("%s: subscriptionID expected, found empty string", test.name)
			}
			if p.tenantID == "" {
				t.Errorf("%s: tenantID expected, found empty string", test.name)
			}
			if p.location == "" {
				t.Errorf("%s: location expected, found empty string", test.name)
			}
			if p.resourceGroup == "" {
				t.Errorf("%s: resourceGroup expected, found empty string", test.name)
			}
			// hostname is always optional to set env var for but we always expect a value
			if p.hostname == "" {
				t.Errorf("%s: hostname expected, found empty string", test.name)
			}
		}
	}
}
