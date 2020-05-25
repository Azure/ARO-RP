package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestPullSecretReconciler(t *testing.T) {
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
	tests := []struct {
		name        string
		request     ctrl.Request
		tokens      map[string]string
		fakecli     *fake.Clientset
		wantErr     bool
		want        string
		wantCreated bool
		wantDeleted bool
		wantUpdated bool
	}{
		{
			name:    "deleted pull secret",
			fakecli: fake.NewSimpleClientset(),
			tokens: map[string]string{
				"arosvc.azurecr.io": "ZnJlZDplbnRlcg==",
			},
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantCreated: true,
		},
		{
			name:    "missing arosvc pull secret",
			fakecli: newFakecli(&v1.Secret{}),
			tokens: map[string]string{
				"arosvc.azurecr.io": "ZnJlZDplbnRlcg==",
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
			tokens: map[string]string{
				"arosvc.azurecr.io": "ZnJlZDplbnRlcg==",
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
			tokens: map[string]string{
				"arosvc.azurecr.io": "ZnJlZDplbnRlcg==",
			},
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantUpdated: true,
		},
		{
			name: "wrong secret type",
			fakecli: newFakecli(&v1.Secret{
				Type: v1.SecretTypeOpaque,
			}),
			tokens: map[string]string{
				"arosvc.azurecr.io": "ZnJlZDplbnRlcg==",
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
			tokens: map[string]string{
				"arosvc.azurecr.io": "ZnJlZDplbnRlcg==",
			},
			want: `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
		},
	}
	for _, tt := range tests {
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

			r := &PullsecretReconciler{
				Kubernetescli:           tt.fakecli,
				Log:                     logrus.NewEntry(logrus.StandardLogger()),
				requiredRepoTokensStore: tt.tokens,
			}
			if tt.request.Name == "" {
				tt.request.NamespacedName = pullSecretName
			}

			_, err := r.Reconcile(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("PullsecretReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
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

			s, err := r.Kubernetescli.CoreV1().Secrets("openshift-config").Get("pull-secret", metav1.GetOptions{})
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
