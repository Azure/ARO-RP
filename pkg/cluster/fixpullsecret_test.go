package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestFixPullSecret(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name        string
		current     []byte
		rps         []*api.RegistryProfile
		want        string
		wantUpdated bool
	}{
		{
			name: "missing pull secret",
			rps: []*api.RegistryProfile{
				{
					Name:     "arosvc.azurecr.io",
					Username: "fred",
					Password: "enter",
				},
			},
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantUpdated: true,
		},
		{
			name:    "modified pull secret",
			current: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":""}}}`),
			rps: []*api.RegistryProfile{
				{
					Name:     "arosvc.azurecr.io",
					Username: "fred",
					Password: "enter",
				},
			},
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantUpdated: true,
		},
		{
			name:    "no change",
			current: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			rps: []*api.RegistryProfile{
				{
					Name:     "arosvc.azurecr.io",
					Username: "fred",
					Password: "enter",
				},
			},
			want: `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var updated bool

			fakecli := fake.NewSimpleClientset(&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pull-secret",
					Namespace: "openshift-config",
				},
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: tt.current,
				},
			})

			fakecli.PrependReactor("update", "secrets", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				updated = true
				return false, nil, nil
			})

			m := &manager{
				kubernetescli: fakecli,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							RegistryProfiles: tt.rps,
						},
					},
				},
			}

			err := m.fixPullSecret(ctx)
			if err != nil {
				t.Error(err)
			}

			if updated != tt.wantUpdated {
				t.Fatal(updated)
			}

			s, err := m.kubernetescli.CoreV1().Secrets("openshift-config").Get(ctx, "pull-secret", metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			if string(s.Data[v1.DockerConfigJsonKey]) != tt.want {
				t.Error(string(s.Data[v1.DockerConfigJsonKey]))
			}
		})
	}
}
