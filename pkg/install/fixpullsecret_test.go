package install

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

	newFakecli := func(s *v1.Secret) *fake.Clientset {
		s.ObjectMeta = metav1.ObjectMeta{
			Name:      "pull-secret",
			Namespace: "openshift-config",
		}

		if s.Type == "" {
			s.Type = v1.SecretTypeDockerConfigJson
		}

		return fake.NewSimpleClientset(s)
	}

	for _, tt := range []struct {
		name        string
		fakecli     *fake.Clientset
		rps         []*api.RegistryProfile
		want        string
		wantCreated bool
		wantDeleted bool
		wantUpdated bool
	}{
		{
			name:    "deleted pull secret",
			fakecli: fake.NewSimpleClientset(),
			rps: []*api.RegistryProfile{
				{
					Name:     "arosvc.azurecr.io",
					Username: "fred",
					Password: "enter",
				},
			},
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantCreated: true,
		},
		{
			name:    "missing arosvc pull secret",
			fakecli: newFakecli(&v1.Secret{}),
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
			name: "modified arosvc pull secret",
			fakecli: newFakecli(&v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":""}}}`),
				},
			}),
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
			name: "unparseable secret",
			fakecli: newFakecli(&v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`bad`),
				},
			}),
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
			name: "wrong secret type",
			fakecli: newFakecli(&v1.Secret{
				Type: v1.SecretTypeOpaque,
			}),
			rps: []*api.RegistryProfile{
				{
					Name:     "arosvc.azurecr.io",
					Username: "fred",
					Password: "enter",
				},
			},
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantCreated: true,
			wantDeleted: true,
		},
		{
			name: "no change",
			fakecli: newFakecli(&v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			}),
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
			var created, deleted, updated bool

			tt.fakecli.PrependReactor("create", "secrets", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				created = true
				return false, nil, nil
			})

			tt.fakecli.PrependReactor("delete", "secrets", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				deleted = true
				return false, nil, nil
			})

			tt.fakecli.PrependReactor("update", "secrets", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				updated = true
				return false, nil, nil
			})

			i := &Installer{
				kubernetescli: tt.fakecli,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							RegistryProfiles: tt.rps,
						},
					},
				},
			}

			err := i.fixPullSecret(ctx)
			if err != nil {
				t.Error(err)
			}

			if created != tt.wantCreated {
				t.Fatal(created)
			}

			if deleted != tt.wantDeleted {
				t.Fatal(deleted)
			}

			if updated != tt.wantUpdated {
				t.Fatal(updated)
			}

			s, err := i.kubernetescli.CoreV1().Secrets("openshift-config").Get("pull-secret", metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			if s.Type != v1.SecretTypeDockerConfigJson {
				t.Error(s.Type)
			}

			if string(s.Data[v1.DockerConfigJsonKey]) != tt.want {
				t.Error(string(s.Data[v1.DockerConfigJsonKey]))
			}
		})
	}
}
