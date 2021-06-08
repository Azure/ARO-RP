package instancemetadata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"
)

func TestProdEnvPopulateInstanceMetadata(t *testing.T) {
	hostname, _ := os.Hostname()

	tests := []struct {
		name        string
		environment map[string]string
		expectErr   bool
		expected    prodfromenv
	}{
		{
			name:        "no environment",
			environment: make(map[string]string),
			expectErr:   true,
			expected:    prodfromenv{},
		},
		{
			name: "env missing all keys",
			environment: map[string]string{
				"someKey": "someValue",
			},
			expectErr: true,
			expected:  prodfromenv{},
		},
		{
			name: "env missing some keys",
			environment: map[string]string{
				"RESOURCEGROUP": "my-rg",
			},
			expectErr: true,
			expected:  prodfromenv{},
		},
		{
			name: "env valid environment",
			environment: map[string]string{
				"AZURE_ENVIRONMENT":     azure.PublicCloud.Name,
				"AZURE_SUBSCRIPTION_ID": "some-sub-guid",
				"AZURE_TENANT_ID":       "some-tenant-guid",
				"LOCATION":              "some-region",
				"RESOURCEGROUP":         "my-resourceGroup",
			},
			expectErr: false,
			expected: prodfromenv{
				instanceMetadata: instanceMetadata{
					environment:    &azure.PublicCloud,
					subscriptionID: "some-sub-guid",
					tenantID:       "some-tenant-guid",
					location:       "some-region",
					resourceGroup:  "my-resourceGroup",
					hostname:       hostname,
				},
			},
		},
		{
			name: "env invalid environment",
			environment: map[string]string{
				"AZURE_ENVIRONMENT":     "ThisEnvDoesNotExist",
				"AZURE_SUBSCRIPTION_ID": "some-sub-guid",
				"AZURE_TENANT_ID":       "some-tenant-guid",
				"LOCATION":              "some-region",
				"RESOURCEGROUP":         "my-resourceGroup",
			},
			expectErr: true,
			expected: prodfromenv{
				instanceMetadata: instanceMetadata{
					environment:    nil,
					subscriptionID: "some-sub-guid",
					tenantID:       "some-tenant-guid",
					location:       "some-region",
					resourceGroup:  "my-resourceGroup",
					hostname:       hostname,
				},
			},
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
			expected: prodfromenv{
				instanceMetadata: instanceMetadata{
					environment:    &azure.PublicCloud,
					subscriptionID: "some-sub-guid",
					tenantID:       "some-tenant-guid",
					location:       "some-region",
					resourceGroup:  "my-resourceGroup",
					hostname:       "my.over.ride",
				},
			},
		},
	}

	for _, test := range tests {
		p := &prodfromenv{
			Getenv: func(key string) string {
				return test.environment[key]
			},
			LookupEnv: func(key string) (string, bool) {
				value, ok := test.environment[key]
				return value, ok
			},
		}
		err := p.populateInstanceMetadata()
		if test.expectErr != (err != nil) {
			t.Errorf("%s: expected error %#v got %#v", test.name, test.expectErr, err)
		} else if !test.expectErr {
			// verify there are values for all required fields
			if p.environment != nil && test.expected.environment != nil {
				pName := ""
				eName := ""
				if p.environment != nil {
					pName = p.environment.Name
				}
				if test.expected.environment != nil {
					eName = test.expected.environment.Name
				}
				if pName != eName {
					t.Errorf("%s: unexpected environment Name value, expected %#v got %#v", test.name, eName, pName)
				}
			} else if p.environment != test.expected.environment {
				// one of these is nil and the other is not
				t.Errorf("%s: unexpected environment value, expected %#v got %#v", test.name, test.expected.environment, p.environment)
			}
			if p.subscriptionID != test.expected.subscriptionID {
				t.Errorf("%s: unexpected subscriptionID value, expected %#v got %#v", test.name, test.expected.subscriptionID, p.subscriptionID)
			}
			if p.tenantID != test.expected.tenantID {
				t.Errorf("%s: unexpected tenantID value, expected %#v got %#v", test.name, test.expected.tenantID, p.tenantID)
			}
			if p.location != test.expected.location {
				t.Errorf("%s: unexpected environment value, expected %#v got %#v", test.name, test.expected.location, p.location)
			}
			if p.resourceGroup != test.expected.resourceGroup {
				t.Errorf("%s: unexpected environment value, expected %#v got %#v", test.name, test.expected.resourceGroup, p.resourceGroup)
			}
			if p.hostname != test.expected.hostname {
				t.Errorf("%s: unexpected environment value, expected %#v got %#v", test.name, test.expected.hostname, p.hostname)
			}
		}
	}
}
