package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestSetRegistryProfiles(t *testing.T) {
	original := `{"auths":{"arosvc.azurecr.io":{"auth":"eDp5"},"registry.redhat.io":{"auth":"eTp6"}}}`

	for _, tt := range []struct {
		name        string
		ps          string
		rps         []*api.RegistryProfile
		want        *pullSecret
		wantChanged bool
		wantErr     string
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
				Auths: map[string]map[string]any{
					"arosvc.azurecr.io": {
						"auth": "ZnJlZDplbnRlcg==",
					},
					"registry.redhat.io": {
						"auth": "eTp6",
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
				Auths: map[string]map[string]any{
					"arosvc.azurecr.io": {
						"auth": "ZnJlZDplbnRlcg==",
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "no change",
			ps:   `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"arointsvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"eTp6"}}}`,
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
				Auths: map[string]map[string]any{
					"arosvc.azurecr.io": {
						"auth": "ZnJlZDplbnRlcg==",
					},
					"registry.redhat.io": {
						"auth": "eTp6",
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
				Auths: map[string]map[string]any{
					"arosvc.azurecr.io": {
						"auth": "eDp5",
					},
					"registry.redhat.io": {
						"auth": "eTp6",
					},
				},
			},
		},
		{
			name:        "invalid base64 in preserved entry",
			ps:          `{"auths":{"byo.example":{"auth":"` + strings.Repeat("A", 236) + `!"}}}`,
			rps:         []*api.RegistryProfile{{Name: "rp.example", Username: "user", Password: "password"}},
			wantChanged: true,
			wantErr:     "invalid pull secret: malformed auth token for key byo.example: invalid Base64",
		},
		{
			name:    "decoded auth without separator",
			ps:      `{"auths":{"byo.example":{"auth":"dXNlcg=="}}}`,
			wantErr: "invalid pull secret: malformed auth token for key byo.example: not in expected user and secret format",
		},
		{
			name:    "non-string auth",
			ps:      `{"auths":{"byo.example":{"auth":123}}}`,
			wantErr: "invalid pull secret: malformed pullsecret (auth must be a string) for key byo.example",
		},
		{
			name:    "null auth",
			ps:      `{"auths":{"byo.example":{"auth":null}}}`,
			wantErr: "invalid pull secret: malformed pullsecret (auth must be a string) for key byo.example",
		},
		{
			name:    "missing auth",
			ps:      `{"auths":{"byo.example":{"email":"user@example.com"}}}`,
			wantErr: "invalid pull secret: malformed pullsecret (no auth key) for key byo.example",
		},
		{
			name:    "null entry",
			ps:      `{"auths":{"byo.example":null}}`,
			wantErr: "invalid pull secret: malformed pullsecret (no auth key) for key byo.example",
		},
		{
			name:    "null secret",
			ps:      `null`,
			wantErr: "invalid pull secret: expected a JSON object",
		},
		{
			name: "overwritten invalid entry",
			ps:   `{"auths":{"rp.example":{"auth":"!"}}}`,
			rps:  []*api.RegistryProfile{{Name: "rp.example", Username: "user", Password: "password"}},
			want: &pullSecret{
				Auths: map[string]map[string]any{
					"rp.example": {"auth": "dXNlcjpwYXNzd29yZA=="},
				},
			},
			wantChanged: true,
		},
		{
			name:    "credential store without inline auth",
			ps:      `{"auths":{"byo.example":{"credsStore":"helper"}}}`,
			wantErr: "invalid pull secret: malformed pullsecret (no auth key) for key byo.example",
		},
		{
			name: "empty secret without profiles",
			want: &pullSecret{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ps, changed, err := SetRegistryProfiles(tt.ps, tt.rps...)
			if changed != tt.wantChanged {
				t.Errorf("got changed %v, want %v", changed, tt.wantChanged)
			}
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("got error %v, want %q", err, tt.wantErr)
				}
				if ps != "" {
					t.Fatalf("got output %q on error", ps)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var got *pullSecret
			if err := json.Unmarshal([]byte(ps), &got); err != nil {
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
				Auths: map[string]map[string]any{
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
				Auths: map[string]map[string]any{
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
				Auths: map[string]map[string]any{
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
				Auths: map[string]map[string]any{
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
				Auths: map[string]map[string]any{
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
				Auths: map[string]map[string]any{
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
				Auths: map[string]map[string]any{
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
				Auths: map[string]map[string]any{
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
