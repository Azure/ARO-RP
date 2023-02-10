package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func TestPullSecretReconciler(t *testing.T) {
	baseCluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     arov1alpha1.ClusterStatus{},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				controllerEnabled: "true",
				controllerManaged: "true",
			},
		},
	}

	tests := []struct {
		name     string
		request  ctrl.Request
		secrets  []client.Object
		instance *arov1alpha1.Cluster
		wantKeys []string
		wantErr  bool
		want     string
	}{
		{
			name: "deleted pull secret",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      operator.SecretName,
						Namespace: operator.Namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`)},
				},
			},
			instance: baseCluster,
			want:     `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys: nil,
		},
		{
			name: "missing arosvc pull secret",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pull-secret",
						Namespace: "openshift-config",
					},
					Type: corev1.SecretTypeDockerConfigJson,
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      operator.SecretName,
						Namespace: operator.Namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`)},
				},
			},
			instance: baseCluster,
			want:     `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys: nil,
		},
		{
			name: "modified arosvc pull secret",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pull-secret",
						Namespace: "openshift-config",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":""}}}`),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      operator.SecretName,
						Namespace: operator.Namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`)},
				},
			},
			instance: baseCluster,
			want:     `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys: nil,
		},
		{
			name: "unparseable secret",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pull-secret",
						Namespace: "openshift-config",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(`bad`)},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      operator.SecretName,
						Namespace: operator.Namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`)},
				},
			},
			instance: baseCluster,
			want:     `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys: nil,
		},
		{
			name: "wrong secret type",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pull-secret",
						Namespace: "openshift-config",
					},
					Type: corev1.SecretTypeOpaque,
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      operator.SecretName,
						Namespace: operator.Namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`)},
				},
			},
			instance: baseCluster,
			want:     `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys: nil,
		},
		{
			name: "no change",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pull-secret",
						Namespace: "openshift-config",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`)},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      operator.SecretName,
						Namespace: operator.Namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`)},
				},
			},
			instance: baseCluster,
			want:     `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys: nil,
		},
		{
			name: "valid RH keys present",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pull-secret",
						Namespace: "openshift-config",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="},"cloud.openshift.com":{"auth":"ZnJlZDplbnRlcg=="}}}`),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      operator.SecretName,
						Namespace: operator.Namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="},"cloud.openshift.com":{"auth":"ZnJlZDplbnRlcg=="}}}`),
					},
				},
			},
			instance: baseCluster,
			want:     `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="},"cloud.openshift.com":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys: []string{"registry.redhat.io", "cloud.openshift.com"},
		},
		{
			name: "management disabled, valid RH key present",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pull-secret",
						Namespace: "openshift-config",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      operator.SecretName,
						Namespace: operator.Namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
					},
				},
			},
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Status:     arov1alpha1.ClusterStatus{},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: "true",
						controllerManaged: "false",
					},
				},
			},
			want:     `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys: []string{"registry.redhat.io"},
		},
		{
			name: "management disabled, valid RH key missing",
			secrets: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pull-secret",
						Namespace: "openshift-config",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`)},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      operator.SecretName,
						Namespace: operator.Namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`)},
				},
			},
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Status:     arov1alpha1.ClusterStatus{},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: "true",
						controllerManaged: "false",
					},
				},
			},
			want:     `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			clientFake := ctrlfake.NewClientBuilder().WithObjects(tt.instance).WithObjects(tt.secrets...).Build()

			r := &Reconciler{
				log:    logrus.NewEntry(logrus.StandardLogger()),
				client: clientFake,
			}
			if tt.request.Name == "" {
				tt.request.NamespacedName = pullSecretName
			}

			_, err := r.Reconcile(ctx, tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("PullsecretReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			s := &corev1.Secret{}
			err = r.client.Get(ctx, types.NamespacedName{Namespace: "openshift-config", Name: "pull-secret"}, s)
			if err != nil {
				t.Error(err)
			}

			if s.Type != corev1.SecretTypeDockerConfigJson {
				t.Errorf("Unexpected secret type: %s", s.Type)
			}

			if string(s.Data[corev1.DockerConfigJsonKey]) != tt.want {
				t.Fatalf("Unexpected secret data.\ngot: %s\nwant: %s", string(s.Data[corev1.DockerConfigJsonKey]), tt.want)
			}

			cluster := &arov1alpha1.Cluster{}
			err = clientFake.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster)
			if err != nil {
				t.Fatal("Error found")
			}

			statusBytes, err := json.Marshal(&cluster.Status)
			if err != nil {
				t.Fatal("Unmarshal expects valid data")
			}

			status := arov1alpha1.ClusterStatus{}
			err = json.Unmarshal(statusBytes, &status)
			if err != nil {
				t.Fatal("Expected to parse status")
			}

			if !reflect.DeepEqual(status.RedHatKeysPresent, tt.wantKeys) {
				t.Fatalf("Unexpected status found\nwant: %v\ngot: %v", tt.wantKeys, status.RedHatKeysPresent)
			}
		})
	}
}

func TestParseRedHatKeys(t *testing.T) {
	test := []struct {
		name        string
		ps          *corev1.Secret
		wantKeys    []string
		wantMissing string
		wantErr     string
	}{
		{
			name: "without rh key",
			ps: &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}},
		},
		{
			name: "with all rh key",
			ps: &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.connect.redhat.com":{"auth":"ZnJlZDplbnRlcg=="},"cloud.openshift.com":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}},
			wantKeys: []string{"registry.redhat.io", "cloud.openshift.com", "registry.connect.redhat.com"},
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}

			out, err := r.parseRedHatKeys(tt.ps)
			if err != nil && err.Error() != tt.wantErr {
				t.Fatalf("Unexpected error:\nwant: %s\ngot: %s", tt.wantErr, err.Error())
			}

			if !reflect.DeepEqual(out, tt.wantKeys) {
				t.Fatalf("Enexpected keys found:\nwant: %v\ngot: %v", tt.wantKeys, out)
			}
		})
	}
}

func TestEnsureGlobalPullSecret(t *testing.T) {
	test := []struct {
		name               string
		initialSecret      *corev1.Secret
		operatorPullSecret *corev1.Secret
		pullSecret         *corev1.Secret
		wantSecret         *corev1.Secret
		wantError          string
	}{
		{
			name: "Red Hat Key present",
			initialSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pull-secret",
					Namespace:       "openshift-config",
					ResourceVersion: "1",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			pullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pull-secret",
					Namespace:       "openshift-config",
					ResourceVersion: "1",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			operatorPullSecret: &corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pull-secret",
					Namespace:       "openshift-config",
					ResourceVersion: "1",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantError: "",
		},
		{
			name: "Red Hat Key missing",
			initialSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pull-secret",
					Namespace:       "openshift-config",
					ResourceVersion: "1",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			pullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pull-secret",
					Namespace: "openshift-config",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			operatorPullSecret: &corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pull-secret",
					Namespace:       "openshift-config",
					ResourceVersion: "2",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
		},
		{
			name: "Red Hat key added should merge in",
			initialSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pull-secret",
					Namespace:       "openshift-config",
					ResourceVersion: "1",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			pullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pull-secret",
					Namespace: "openshift-config",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			operatorPullSecret: &corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pull-secret",
					Namespace:       "openshift-config",
					ResourceVersion: "2",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
		},
		{
			name: "Pull secret empty",
			initialSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pull-secret",
					Namespace:       "openshift-config",
					ResourceVersion: "1",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			pullSecret: &corev1.Secret{},
			operatorPullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pull-secret",
					Namespace: "openshift-config",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            pullSecretName.Name,
					Namespace:       pullSecretName.Namespace,
					ResourceVersion: "1",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
		},
		{
			name:          "Secret missing",
			initialSecret: nil,
			pullSecret:    nil,
			operatorPullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            pullSecretName.Name,
					Namespace:       pullSecretName.Namespace,
					ResourceVersion: "1",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            pullSecretName.Name,
					Namespace:       pullSecretName.Namespace,
					ResourceVersion: "1",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
		},
		{
			name: "Red Hat Key present but secret type broken",
			initialSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pull-secret",
					Namespace:       "openshift-config",
					ResourceVersion: "1",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			pullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeBasicAuth,
			},
			operatorPullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            pullSecretName.Name,
					Namespace:       pullSecretName.Namespace,
					ResourceVersion: "1",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			wantError: "",
		},
		{
			name: "Secret auth key broken broken",
			initialSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pull-secret",
					Namespace:       "openshift-config",
					ResourceVersion: "1",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			pullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"lbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			operatorPullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            pullSecretName.Name,
					Namespace:       pullSecretName.Namespace,
					ResourceVersion: "2",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
		},
		{
			name: "Secret not parseable",
			initialSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pull-secret",
					Namespace:       "openshift-config",
					ResourceVersion: "1",
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			pullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`bad`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			operatorPullSecret: &corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            pullSecretName.Name,
					Namespace:       pullSecretName.Namespace,
					ResourceVersion: "2",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
		},
		{
			name: "Operator secret not parseable",
			initialSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pull-secret",
					Namespace:       "openshift-config",
					ResourceVersion: "1",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`bad`),
				},
			},
			pullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`bad`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			operatorPullSecret: &corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`bad`),
				},
			},
			wantSecret: nil,
			wantError:  "invalid character 'b' looking for beginning of value",
		},
		{
			name: "Operator secret nil",
			initialSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pull-secret",
					Namespace:       "openshift-config",
					ResourceVersion: "1",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`bad`),
				},
			},
			pullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`bad`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			operatorPullSecret: nil,
			wantSecret:         nil,
			wantError:          "nil operator secret, cannot verify userData integrity",
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			clientBuilder := ctrlfake.NewClientBuilder()
			if tt.initialSecret != nil {
				clientBuilder = clientBuilder.WithObjects(tt.initialSecret)
			}

			r := &Reconciler{
				client: clientBuilder.Build(),
			}

			s, err := r.ensureGlobalPullSecret(ctx, tt.operatorPullSecret, tt.pullSecret)
			if err != nil && (err.Error() != tt.wantError) {
				t.Fatalf("Unexpected error\ngot: %s\nwant: %s", err.Error(), tt.wantError)
			}

			if !reflect.DeepEqual(s, tt.wantSecret) {
				t.Fatalf(cmp.Diff(s, tt.wantSecret))
			}
		})
	}
}
