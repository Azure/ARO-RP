package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestSetRegistryProfiles(t *testing.T) {
	original := `{"auths":{"arosvc.azurecr.io":{"auth":"x"},"registry.redhat.io":{"auth":"y"}}}`

	for _, tt := range []struct {
		name string
		ps   string
		rp   *api.RegistryProfile
		want *pullSecret
	}{
		{
			name: "replace",
			ps:   original,
			rp: &api.RegistryProfile{
				Name:     "arosvc.azurecr.io",
				Username: "fred",
				Password: "enter",
			},
			want: &pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "ZnJlZDplbnRlcg==",
					},
					"registry.redhat.io": {
						"auth": "y",
					},
				},
			},
		},
		{
			name: "add",
			ps:   original,
			rp: &api.RegistryProfile{
				Name:     "arosvc-int.azurecr.io",
				Username: "fred",
				Password: "enter",
			},
			want: &pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "x",
					},
					"registry.redhat.io": {
						"auth": "y",
					},
					"arosvc-int.azurecr.io": {
						"auth": "ZnJlZDplbnRlcg==",
					},
				},
			},
		},
		{
			name: "new",
			rp: &api.RegistryProfile{
				Name:     "arosvc.azurecr.io",
				Username: "fred",
				Password: "enter",
			},
			want: &pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "ZnJlZDplbnRlcg==",
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ps, err := SetRegistryProfiles(tt.ps, tt.rp)
			if err != nil {
				t.Fatal(err)
			}

			var got *pullSecret
			err = json.Unmarshal([]byte(ps), &got)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Error(ps)
			}
		})
	}
}
