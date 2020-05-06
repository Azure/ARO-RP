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
		name        string
		ps          string
		rps         []*api.RegistryProfile
		want        *pullSecret
		wantChanged bool
	}{
		{
			name: "add and replace",
			ps:   original,
			rps: []*api.RegistryProfile{
				{
					Name:     "arosvc.azurecr.io",
					Username: "fred",
					Password: "enter",
				},
				{
					Name:     "arosvc-int.azurecr.io",
					Username: "fred",
					Password: "enter",
				},
			},
			want: &pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "ZnJlZDplbnRlcg==",
					},
					"registry.redhat.io": {
						"auth": "y",
					},
					"arosvc-int.azurecr.io": {
						"auth": "ZnJlZDplbnRlcg==",
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "new",
			rps: []*api.RegistryProfile{
				{
					Name:     "arosvc.azurecr.io",
					Username: "fred",
					Password: "enter",
				},
			},
			want: &pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "ZnJlZDplbnRlcg==",
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "no change",
			ps:   `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"arosvc-int.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"y"}}}`,
			rps: []*api.RegistryProfile{
				{
					Name:     "arosvc.azurecr.io",
					Username: "fred",
					Password: "enter",
				},
				{
					Name:     "arosvc-int.azurecr.io",
					Username: "fred",
					Password: "enter",
				},
			},
			want: &pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "ZnJlZDplbnRlcg==",
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			ps, changed, err := SetRegistryProfiles(tt.ps, tt.rps...)
			if err != nil {
				t.Fatal(err)
			}

			if changed != tt.wantChanged {
				t.Error(changed)
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
