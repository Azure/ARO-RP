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

	newFakecli := func(data map[string][]byte) *fake.Clientset {
		return fake.NewSimpleClientset(&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pull-secret",
				Namespace: "openshift-config",
			},
			Data: data,
		})
	}

	for _, tt := range []struct {
		name        string
		fakecli     *fake.Clientset
		rps         []*api.RegistryProfile
		want        string
		wantCreated bool
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
			fakecli: newFakecli(nil),
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
			fakecli: newFakecli(map[string][]byte{
				v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":""}}}`),
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
			name: "no change",
			fakecli: newFakecli(map[string][]byte{
				v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
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
			var created, updated bool

			tt.fakecli.PrependReactor("create", "secrets", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				created = true
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

			if updated != tt.wantUpdated {
				t.Fatal(updated)
			}

			s, err := i.kubernetescli.CoreV1().Secrets("openshift-config").Get("pull-secret", metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			if string(s.Data[v1.DockerConfigJsonKey]) != tt.want {
				t.Error(string(s.Data[v1.DockerConfigJsonKey]))
			}
		})
	}
}
