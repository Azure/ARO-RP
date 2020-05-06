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

func TestRedacted(t *testing.T) {
	tests := []struct {
		name    string
		ps      string
		want    string
		wantErr bool
	}{
		{
			name: "normal",
			ps:   `{"auths":{"arosvc.azurecr.io":{"auth":"x"},"registry.redhat.io":{"auth":"y"}}}`,
			want: `{"auths":{"arosvc.azurecr.io":{"auth":"#redacted"},"registry.redhat.io":{"auth":"#redacted"}}}`,
		},
		{
			name: "auth empty",
			ps:   `{"auths":{"arosvc.azurecr.io":{"auth":""},"registry.redhat.io":{"auth":"y"}}}`,
			want: `{"auths":{"arosvc.azurecr.io":{"auth":""},"registry.redhat.io":{"auth":"#redacted"}}}`,
		},
		{
			name: "missing auth",
			ps:   `{"auths":{"arosvc.azurecr.io":{},"registry.redhat.io":{"auth":"y"}}}`,
			want: `{"auths":{"arosvc.azurecr.io":{},"registry.redhat.io":{"auth":"#redacted"}}}`,
		},
		{
			name: "just brackets",
			ps:   `{}`,
			want: ``,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Redacted(tt.ps)
			if (err != nil) != tt.wantErr {
				t.Errorf("Redacted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Redacted() = %v, want %v", got, tt.want)
			}
		})
	}
}
