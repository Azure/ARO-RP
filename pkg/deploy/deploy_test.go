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
		name   string
		ps     map[string]interface{}
		config Configuration
		want   arm.Parameters
	}{
		{
			name: "no parameters",
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"fullDeploy": {
						Value: false,
					},
				},
			},
		},
		{
			name: "valid",
			ps: map[string]interface{}{
				"adminApiCaBundle":                   nil,
				"databaseAccountName":                nil,
				"extraClusterKeyvaultAccessPolicies": nil,
			},
			config: Configuration{
				DatabaseAccountName:                "databaseAccountName",
				ExtraClusterKeyvaultAccessPolicies: []interface{}{"a", 1},
			},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"adminApiCaBundle": {
						Value: "",
					},
					"databaseAccountName": {
						Value: "databaseAccountName",
					},
					"extraClusterKeyvaultAccessPolicies": {
						Value: []interface{}{"a", 1},
					},
					"fullDeploy": {
						Value: false,
					},
				},
			},
		},
		{
			name: "nil slice",
			ps: map[string]interface{}{
				"extraClusterKeyvaultAccessPolicies": nil,
			},
			config: Configuration{},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"extraClusterKeyvaultAccessPolicies": {
						Value: []interface{}(nil),
					},
					"fullDeploy": {
						Value: false,
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := deployer{
				config: &RPConfig{Configuration: &tt.config},
			}

			got := d.getParameters(tt.ps)
			if !reflect.DeepEqual(got, &tt.want) {
				t.Errorf("%#v", got)
			}
		})
	}
}
