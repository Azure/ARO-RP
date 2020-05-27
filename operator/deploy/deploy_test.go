package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestOperatorSetAcrToken(t *testing.T) {
	tests := []struct {
		name          string
		reg           *api.RegistryProfile
		overrideToken string
		expect        map[string]string
	}{
		{
			name:   "normal dev, no PULL_SECRET_PATH",
			reg:    &api.RegistryProfile{Name: "foo"},
			expect: map[string]string{},
		},
		{
			name:          "dev with PULL_SECRET_PATH",
			reg:           &api.RegistryProfile{Name: "foo"},
			overrideToken: "thisplease",
			expect:        map[string]string{"foo.azurecr.io": "thisplease"},
		},
		{
			name: "prod",
			reg: &api.RegistryProfile{
				Name:     "foo.azurecr.io",
				Username: "bla",
				Password: "letmein",
			},
			expect: map[string]string{"foo.azurecr.io": "bla:letmein"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &operator{
				acrRegName: "foo.azurecr.io",
				acrName:    "foo",
			}
			oc := &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					RegistryProfiles: []*api.RegistryProfile{tt.reg},
				},
			}

			overridePath := ""
			var err error
			if tt.overrideToken != "" {
				overridePath, err = ioutil.TempDir("", "pullsecret")
				if err != nil {
					t.Error(err)
				}
				defer os.RemoveAll(overridePath)
				err = ioutil.WriteFile(path.Join(overridePath, o.acrRegName), []byte(tt.overrideToken), 0644)
				if err != nil {
					t.Error(err)
				}
			}

			o.setAcrToken(oc, overridePath)
			if !reflect.DeepEqual(o.regTokens, tt.expect) {
				t.Error(o.regTokens)
			}
		})
	}
}
