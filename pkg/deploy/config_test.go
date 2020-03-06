package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func TestConfigurationFieldParityTest(t *testing.T) {
	var t1, t2 *arm.Parameters
	err := json.Unmarshal(MustAsset(generator.FileRPProductionParameters), &t1)
	if err != nil {
		t.Error(err)
	}
	err = json.Unmarshal(MustAsset(generator.FileRPProductionPredeployParameters), &t2)
	if err != nil {
		t.Error(err)
	}

	// construct map with all parameters in the generated templates
	p := make(map[string]bool, len(t1.Parameters)+len(t2.Parameters))
	for _, t := range []*arm.Parameters{t1, t2} {
		for name := range t.Parameters {
			p[name] = false
		}
	}

	s := reflect.TypeOf(Configuration{})
	for tag := range p {
		for i := 0; i < s.NumField(); i++ {
			field := s.Field(i)
			if field.Tag.Get("json") == tag {
				p[tag] = true
			}
		}
	}

	for tag, exist := range p {
		if !exist {
			t.Fatalf("field %s not found in config.Configuration but exist in templates", tag)
		}
	}
}

func TestMergeConfig(t *testing.T) {
	for _, tt := range []struct {
		name      string
		primary   func(*Configuration)
		secondary func(*Configuration)
		expected  func(*Configuration)
	}{
		{
			name:      "noop",
			primary:   func(p *Configuration) {},
			secondary: func(p *Configuration) {},
			expected:  func(p *Configuration) {},
		},
		{
			name: "primary is not provided",
			primary: func(p *Configuration) {
				p = &Configuration{}
			},
			expected: func(p *Configuration) {},
		},
		{
			name: "primary is slim",
			primary: func(p *Configuration) {
				p.AdminAPICaBundle = ""
				p.AdminAPIClientCertCommonName = ""
				p.ExtraCosmosDBIPs = ""
				p.FPServicePrincipalID = ""
				p.DatabaseAccountName = "database-custom-name"
				p.DomainName = "example.com"
				p.KeyvaultPrefix = "custom-prefix"
			},
			secondary: func(p *Configuration) {},
			expected: func(p *Configuration) {
				p.DatabaseAccountName = "database-custom-name"
				p.DomainName = "example.com"
				p.KeyvaultPrefix = "custom-prefix"
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			primary := &Configuration{}
			if tt.primary != nil {
				tt.primary(primary)
			}

			secondary := &Configuration{}
			if tt.secondary != nil {
				tt.secondary(secondary)
			}

			expected := &Configuration{}
			if tt.expected != nil {
				tt.expected(expected)
			}

			got, err := mergeConfig(primary, secondary)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(expected, got) {
				t.Fatalf("\nexpected:\n%v \ngot:\n%v", expected, &got)
			}

		})
	}
}
