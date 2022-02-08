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
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
)

func TestPullSecretReconciler(t *testing.T) {
	newFakecli := func(s *corev1.Secret, c *corev1.Secret) *fake.Clientset {
		c.ObjectMeta = metav1.ObjectMeta{
			Name:      operator.SecretName,
			Namespace: operator.Namespace,
		}
		c.Type = corev1.SecretTypeOpaque
		if s == nil {
			return fake.NewSimpleClientset(c)
		}

		s.ObjectMeta = metav1.ObjectMeta{
			Name:      "pull-secret",
			Namespace: "openshift-config",
		}
		if s.Type == "" {
			s.Type = corev1.SecretTypeDockerConfigJson
		}
		return fake.NewSimpleClientset(s, c)
	}

	newFakeAro := func(a *arov1alpha1.Cluster) *arofake.Clientset {
		return arofake.NewSimpleClientset(a)
	}

	baseCluster := arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     arov1alpha1.ClusterStatus{},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				ENABLED: "true",
				MANAGED: "true",
			},
		},
	}

	tests := []struct {
		name        string
		request     ctrl.Request
		fakecli     *fake.Clientset
		arocli      *arofake.Clientset
		wantKeys    []string
		wantErr     bool
		want        string
		wantCreated bool
		wantDeleted bool
		wantUpdated bool
	}{
		{
			name: "deleted pull secret",
			fakecli: newFakecli(nil, &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			arocli:      newFakeAro(&baseCluster),
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys:    nil,
			wantCreated: true,
			wantDeleted: true,
		},
		{
			name: "missing arosvc pull secret",
			fakecli: newFakecli(&corev1.Secret{}, &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			arocli:      newFakeAro(&baseCluster),
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys:    nil,
			wantCreated: true,
			wantDeleted: true,
		},
		{
			name: "modified arosvc pull secret",
			fakecli: newFakecli(&corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":""}}}`),
				},
			}, &corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				}}),
			arocli:      newFakeAro(&baseCluster),
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys:    nil,
			wantUpdated: true,
		},
		{
			name: "unparseable secret",
			fakecli: newFakecli(&corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`bad`),
				},
			}, &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			arocli:      newFakeAro(&baseCluster),
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys:    nil,
			wantUpdated: true,
		},
		{
			name: "wrong secret type",
			fakecli: newFakecli(&corev1.Secret{
				Type: corev1.SecretTypeOpaque,
			}, &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			arocli:      newFakeAro(&baseCluster),
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys:    nil,
			wantCreated: true,
			wantDeleted: true,
		},
		{
			name: "no change",
			fakecli: newFakecli(&corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			}, &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			arocli:   newFakeAro(&baseCluster),
			want:     `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys: nil,
		},
		{
			name: "valid RH keys present",
			fakecli: newFakecli(&corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="},"cloud.redhat.com":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			}, &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="},"cloud.redhat.com":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			arocli:   newFakeAro(&baseCluster),
			want:     `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="},"cloud.redhat.com":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys: []string{"registry.redhat.io", "cloud.redhat.com"},
		},
		{
			name: "management disabled, valid RH key present",
			fakecli: newFakecli(&corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			}, &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			arocli: newFakeAro(
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status:     arov1alpha1.ClusterStatus{},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							ENABLED: "true",
							MANAGED: "false",
						},
					},
				}),
			want:     `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys: []string{"registry.redhat.io"},
		},
		{
			name: "management disabled, valid RH key missing",
			fakecli: newFakecli(&corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			}, &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			arocli: newFakeAro(
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status:     arov1alpha1.ClusterStatus{},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							ENABLED: "true",
							MANAGED: "false",
						},
					},
				}),
			want:     `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantKeys: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fakecli.PrependReactor("create", "secrets", func(action ktesting.Action) (handled bool, ret kruntime.Object, err error) {
				if !tt.wantCreated {
					t.Fatal("Unexpected create")
				}
				return false, nil, nil
			})

			tt.fakecli.PrependReactor("delete", "secrets", func(action ktesting.Action) (handled bool, ret kruntime.Object, err error) {
				if !tt.wantDeleted {
					t.Fatalf("Unexpected delete")
				}
				return false, nil, nil
			})

			tt.fakecli.PrependReactor("update", "secrets", func(action ktesting.Action) (handled bool, ret kruntime.Object, err error) {
				if !tt.wantUpdated {
					t.Fatalf("Unexpected update")
				}
				return false, nil, nil
			})

			r := &Reconciler{
				kubernetescli: tt.fakecli,
				log:           logrus.NewEntry(logrus.StandardLogger()),
				arocli:        tt.arocli,
			}
			if tt.request.Name == "" {
				tt.request.NamespacedName = pullSecretName
			}

			_, err := r.Reconcile(context.Background(), tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("PullsecretReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			s, err := r.kubernetescli.CoreV1().Secrets("openshift-config").Get(context.Background(), "pull-secret", metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			if s.Type != corev1.SecretTypeDockerConfigJson {
				t.Errorf("Unexpected secret type: %s", s.Type)
			}

			if string(s.Data[corev1.DockerConfigJsonKey]) != tt.want {
				t.Fatalf("Unexpected secret data.\ngot: %s\nwant: %s", string(s.Data[corev1.DockerConfigJsonKey]), tt.want)
			}

			cluster, err := r.arocli.AroV1alpha1().Clusters().Get(context.Background(), arov1alpha1.SingletonClusterName, metav1.GetOptions{})
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
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.connect.redhat.com":{"auth":"ZnJlZDplbnRlcg=="},"cloud.redhat.com":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}},
			wantKeys: []string{"registry.redhat.io", "cloud.redhat.com", "registry.connect.redhat.com"},
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
	newFakecli := func(s *corev1.Secret) *fake.Clientset {
		if s == nil {
			return fake.NewSimpleClientset()
		}

		s.ObjectMeta = metav1.ObjectMeta{
			Name:      "pull-secret",
			Namespace: "openshift-config",
		}
		if s.Type == "" {
			s.Type = corev1.SecretTypeDockerConfigJson
		}
		return fake.NewSimpleClientset(s)
	}

	test := []struct {
		name               string
		fakecli            *fake.Clientset
		operatorPullSecret *corev1.Secret
		pullSecret         *corev1.Secret
		wantSecret         *corev1.Secret
		wantError          string
	}{
		{
			name: "Red Hat Key present",
			fakecli: newFakecli(&corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			pullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pull-secret",
					Namespace: "openshift-config",
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
					Name:      "pull-secret",
					Namespace: "openshift-config",
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
			fakecli: newFakecli(&corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
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
					Name:      "pull-secret",
					Namespace: "openshift-config",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
		},
		{
			name: "Red Hat key added should merge in",
			fakecli: newFakecli(&corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
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
					Name:      "pull-secret",
					Namespace: "openshift-config",
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
		},
		{
			name: "Secret empty",
			fakecli: newFakecli(&corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
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
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
		},
		{
			name:       "Secret missing",
			fakecli:    newFakecli(nil),
			pullSecret: nil,
			operatorPullSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
		},
		{
			name: "Red Hat Key present but secret type broken",
			fakecli: newFakecli(&corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
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
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
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
			fakecli: newFakecli(&corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
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
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
		},
		{
			name:    "Secret not parseable",
			fakecli: newFakecli(&corev1.Secret{}),
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
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
		},
		{
			name: "Operator secret not parseable",
			fakecli: newFakecli(&corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`bad`),
			}}),
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
			fakecli: newFakecli(&corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`bad`),
			}}),
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
			r := &Reconciler{
				kubernetescli: tt.fakecli,
			}

			s, err := r.ensureGlobalPullSecret(context.Background(), tt.operatorPullSecret, tt.pullSecret)
			if err != nil && (err.Error() != tt.wantError) {
				t.Fatalf("Unexpected error\ngot: %s\nwant: %s", err.Error(), tt.wantError)
			}

			if !reflect.DeepEqual(s, tt.wantSecret) {
				t.Fatalf("Unexpected secret mismatch\ngot: %v\nwant: %v", s, tt.wantSecret)
			}

		})
	}
}
