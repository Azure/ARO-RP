package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func TestGetParameters(t *testing.T) {
	for _, tt := range []struct {
		name        string
		template    map[string]interface{}
		expected    arm.Parameters
		config      func(*RPConfig)
		expectedErr error
	}{
		{
			name:     "no parameters",
			template: map[string]interface{}{},
			expected: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{},
			},
		},
		{
			name: "empty json array",
			template: map[string]interface{}{
				"parameters": map[string]interface{}{
					"extraKeyvaultAccessPolicies": map[string]interface{}{
						"type": "array",
					},
				},
			},
			config: func(c *RPConfig) {
				c.Configuration.ExtraKeyvaultAccessPolicies = []interface{}{"[]"}
			},
			expected: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"extraKeyvaultAccessPolicies": {
						Value: []interface{}{"[]"},
					},
				},
			},
		},
		{
			name: "valid json array",
			template: map[string]interface{}{
				"parameters": map[string]interface{}{
					"extraKeyvaultAccessPolicies": map[string]interface{}{
						"type": "array",
					},
				},
			},
			config: func(c *RPConfig) {
				c.Configuration.ExtraKeyvaultAccessPolicies = []interface{}{`[{"objectId":"bc46cb3a-xxxx-xxxx-xxxx-d2659a531a33"}]`}
			},
			expected: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"extraKeyvaultAccessPolicies": {
						Value: []interface{}{`[{"objectId":"bc46cb3a-xxxx-xxxx-xxxx-d2659a531a33"}]`},
					},
				},
			},
		},
		{
			name: "empty string",
			template: map[string]interface{}{
				"parameters": map[string]interface{}{
					"databaseAccountName": map[string]interface{}{
						"type": "string",
					},
				},
			},
			config: func(c *RPConfig) {
				c.Configuration.DatabaseAccountName = ""
			},
			expected: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"databaseAccountName": {
						Value: "",
					},
				},
			},
		},
		{
			name: "valid string",
			template: map[string]interface{}{
				"parameters": map[string]interface{}{
					"databaseAccountName": map[string]interface{}{
						"type": "string",
					},
				},
			},
			config: func(c *RPConfig) {
				c.Configuration.DatabaseAccountName = "databaseAccountName"
			},
			expected: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"databaseAccountName": {
						Value: "databaseAccountName",
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			config := &RPConfig{
				Configuration: Configuration{
					AdminAPICaBundle: "adminAPICaBundle",
				},
			}
			if tt.config != nil {
				tt.config(config)
			}
			d := deployer{
				config: config,
			}

			got, err := d.getParameters(tt.template)
			// we test either error case or output.
			if err != nil {
				if err.Error() != tt.expectedErr.Error() {
					t.Fatalf("\nexpected:\n%v \ngot:\n%v", tt.expectedErr, err)
					t.FailNow()
				}
			} else {
				if !reflect.DeepEqual(got, &tt.expected) {
					t.Fatalf("\nexpected:\n%v \ngot:\n%v", tt.expected, *got)
				}
			}

		})
	}

}
