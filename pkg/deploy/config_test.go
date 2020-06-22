package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

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
				DatabaseAccountName:    "primary accountname",
				FPServerCertCommonName: "primary fpcert",
			},
			secondary: Configuration{
				FPServerCertCommonName: "secondary fpcert",
				KeyvaultPrefix:         "secondary kv",
			},
			want: Configuration{
				DatabaseAccountName:    "primary accountname",
				FPServerCertCommonName: "primary fpcert",
				KeyvaultPrefix:         "secondary kv",
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
