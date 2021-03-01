package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"

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

func TestFixPullSecretData(t *testing.T) {

	for _, tt := range []struct {
		name          string
		operatorData  SerializedAuthMap
		userData      SerializedAuthMap
		wantFixedData SerializedAuthMap
		wantFixed     bool
	}{
		{
			name: "both the same",
			operatorData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
					"registry.redhat.io": {
						Auth: "y",
					},
				},
			},
			userData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
					"registry.redhat.io": {
						Auth: "y",
					},
				},
			},
			wantFixedData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
					"registry.redhat.io": {
						Auth: "y",
					},
				},
			},
			wantFixed: false,
		},
		{
			name:         "operator empty",
			operatorData: SerializedAuthMap{},
			userData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
					"registry.redhat.io": {
						Auth: "y",
					},
				},
			},
			wantFixedData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
					"registry.redhat.io": {
						Auth: "y",
					},
				},
			},
			wantFixed: false,
		},
		{
			name: "user empty",
			operatorData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
					"registry.redhat.io": {
						Auth: "y",
					},
				},
			},
			userData: SerializedAuthMap{},
			wantFixedData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
					"registry.redhat.io": {
						Auth: "y",
					},
				},
			},
			wantFixed: true,
		},
		{
			name: "user add new auth",
			operatorData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
					"registry.redhat.io": {
						Auth: "y",
					},
				},
			},
			userData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
					"registry.redhat.io": {
						Auth: "y",
					},
					"quay.io": {
						Auth: "z",
					},
				},
			},
			wantFixedData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
					"registry.redhat.io": {
						Auth: "y",
					},
					"quay.io": {
						Auth: "z",
					},
				},
			},
			wantFixed: false,
		},
		{
			name: "user removed aro auth",
			operatorData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
				},
			},
			userData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"registry.redhat.io": {
						Auth: "y",
					},
				},
			},
			wantFixedData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
					"registry.redhat.io": {
						Auth: "y",
					},
				},
			},
			wantFixed: true,
		},
		{
			name: "user changed key",
			operatorData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
				},
			},
			userData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"registry.redhat.io": {
						Auth: "y",
					},
					"arosvc.azurecr.io": {
						Auth: "z",
					},
				},
			},
			wantFixedData: SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
					"registry.redhat.io": {
						Auth: "y",
					},
				},
			},
			wantFixed: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixedData, fixed := FixPullSecretData(&tt.operatorData, &tt.userData)

			if fixed != tt.wantFixed {
				t.Errorf("Want changed: %t, got: %t", tt.wantFixed, fixed)
			}

			if !reflect.DeepEqual(*fixedData, tt.wantFixedData) {
				t.Errorf("Result fixedData is not expected \ngot: %v\nwant:%v", fixedData, tt.wantFixedData)
			}
		})
	}
}

func TestUnmarshalSecret(t *testing.T) {
	test := []struct {
		name      string
		rawData   string
		wantData  *SerializedAuthMap
		wantError string
	}{
		{
			name:    "can unmarshal valid data",
			rawData: `{"auths":{"arosvc.azurecr.io":{"auth":"x"},"registry.redhat.io":{"auth":"y"}}}`,
			wantData: &SerializedAuthMap{
				Auths: map[string]SerializedAuth{
					"arosvc.azurecr.io": {
						Auth: "x",
					},
					"registry.redhat.io": {
						Auth: "y",
					},
				},
			},
			wantError: "",
		},
		{
			name:      "error when invalid data is present",
			rawData:   `{"auths:{"arosvc.azurecr.io":{"auth":"x","registry.redhat.io"{"auth":"y"}}}`,
			wantData:  nil,
			wantError: "invalid character 'a' after object key",
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UnmarshalSecretData(&v1.Secret{Data: map[string][]byte{v1.DockerConfigJsonKey: []byte(tt.rawData)}})
			if err != nil && err.Error() != tt.wantError {
				t.Errorf("Unexpected error: %s", err.Error())
			}
			if !reflect.DeepEqual(result, tt.wantData) {
				t.Errorf("Unexpected unmarshalled data: \ngot: %v\nwant: %v", result, tt.wantData)
			}

		})
	}
}
