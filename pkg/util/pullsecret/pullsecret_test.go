package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"

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
					Name:     "arointsvc.azurecr.io",
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
					"arointsvc.azurecr.io": {
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
			ps:   `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"arointsvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"y"}}}`,
			rps: []*api.RegistryProfile{
				{
					Name:     "arosvc.azurecr.io",
					Username: "fred",
					Password: "enter",
				},
				{
					Name:     "arointsvc.azurecr.io",
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
					"arointsvc.azurecr.io": {
						"auth": "ZnJlZDplbnRlcg==",
					},
				},
			},
		},
		{
			name: "delete",
			ps:   original,
			rps:  []*api.RegistryProfile{},
			want: &pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "x",
					},
					"registry.redhat.io": {
						"auth": "y",
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

func TestMerge(t *testing.T) {
	originalPS := `{"auths":{"arosvc.azurecr.io":{"auth":"x"},"registry.redhat.io":{"auth":"y"}}}`

	for _, tt := range []struct {
		name        string
		base        string
		ps          string
		wantPS      pullSecret
		wantChanged bool
		wantError   error
	}{
		{
			name: "both the same",
			ps:   originalPS,
			base: originalPS,
			wantPS: pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "x",
					},
					"registry.redhat.io": {
						"auth": "y",
					},
				},
			},
			wantChanged: false,
			wantError:   nil,
		},
		{
			name: "ps empty",
			ps:   ``,
			base: originalPS,
			wantPS: pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "x",
					},
					"registry.redhat.io": {
						"auth": "y",
					},
				},
			},
			wantChanged: false,
			wantError:   nil,
		},
		{
			name: "base empty",
			ps:   originalPS,
			base: ``,
			wantPS: pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "x",
					},
					"registry.redhat.io": {
						"auth": "y",
					},
				},
			},
			wantChanged: true,
			wantError:   nil,
		},
		{
			name: "ps add new",
			ps:   `{"auths":{"arosvc.azurecr.io":{"auth":"x"},"registry.redhat.io":{"auth":"y"},"aroacr.azure.io":{"auth":"z"}}}`,
			base: originalPS,
			wantPS: pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "x",
					},
					"registry.redhat.io": {
						"auth": "y",
					},
					"aroacr.azure.io": {
						"auth": "z",
					},
				},
			},
			wantChanged: true,
			wantError:   nil,
		},
		{
			name: "ps remove one",
			base: `{"auths":{"arosvc.azurecr.io":{"auth":"x"},"registry.redhat.io":{"auth":"y"},"aroacr.azure.io":{"auth":"z"}}}`,
			ps:   originalPS,
			wantPS: pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "x",
					},
					"registry.redhat.io": {
						"auth": "y",
					},
					"aroacr.azure.io": {
						"auth": "z",
					},
				},
			},
			wantChanged: false,
			wantError:   nil,
		},
		{
			name: "ps change key one",
			ps:   `{"auths":{"arosvc.azurecr.io":{"auth":"a"},"registry.redhat.io":{"auth":"y"}}}`,
			base: originalPS,
			wantPS: pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "a",
					},
					"registry.redhat.io": {
						"auth": "y",
					},
				},
			},
			wantChanged: true,
			wantError:   nil,
		},
		{
			name: "base add new",
			base: `{"auths":{"arosvc.azurecr.io":{"auth":"x"},"registry.redhat.io":{"auth":"y"},"aroacr.azure.io":{"auth":"z"}}}`,
			ps:   originalPS,
			wantPS: pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "x",
					},
					"registry.redhat.io": {
						"auth": "y",
					},
					"aroacr.azure.io": {
						"auth": "z",
					},
				},
			},
			wantChanged: false,
			wantError:   nil,
		},
		{
			name: "base delete one",
			base: `{"auths":{"arosvc.azurecr.io":{"auth":"x"}}}`,
			ps:   originalPS,
			wantPS: pullSecret{
				Auths: map[string]map[string]interface{}{
					"arosvc.azurecr.io": {
						"auth": "x",
					},
					"registry.redhat.io": {
						"auth": "y",
					},
				},
			},
			wantChanged: true,
			wantError:   nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ps, changed, err := Merge(tt.base, tt.ps)
			if err != tt.wantError {
				t.Fatalf("Want err: %s, Got: %s", tt.wantError.Error(), err.Error())
			}

			if changed != tt.wantChanged {
				t.Fatalf("Want changed: %t, got: %t", tt.wantChanged, changed)
			}

			var got pullSecret
			err = json.Unmarshal([]byte(ps), &got)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, tt.wantPS) {
				t.Errorf("wrong ps: %s", ps)
			}
		})
	}
}

func TestUnmarshalSecretData(t *testing.T) {
	test := []struct {
		name     string
		ps       *corev1.Secret
		wantAuth map[string]string
		wantErr  string
	}{
		{
			name: "ok secret",
			ps: &corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}, "registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantAuth: map[string]string{
				"arosvc.azurecr.io":  "ZnJlZDplbnRlcg==",
				"registry.redhat.io": "ZnJlZDplbnRlcg==",
			},
		},
		{
			name: "bad secret",
			ps: &corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{}, "registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantAuth: map[string]string{
				"registry.redhat.io": "ZnJlZDplbnRlcg==",
			},
		},
		{
			name: "broken secret",
			ps: &corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantErr: "invalid character ':' after object key:value pair",
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			out, err := UnmarshalSecretData(tt.ps)
			if err != nil {
				if err.Error() != tt.wantErr {
					t.Fatal(err.Error())
				}
			} else if !reflect.DeepEqual(out, tt.wantAuth) {
				t.Fatalf("Auth does not match:\n%v\n%v", out, tt.wantAuth)
			}
		})
	}
}
