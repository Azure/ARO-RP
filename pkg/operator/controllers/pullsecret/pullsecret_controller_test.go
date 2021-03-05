package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/status"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

	baseCluster := newFakeAro(
		&arov1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
			Status:     arov1alpha1.ClusterStatus{},
		})

	tests := []struct {
		name           string
		request        ctrl.Request
		fakecli        *fake.Clientset
		arocli         *arofake.Clientset
		wantConditions []status.Condition
		wantErr        bool
		want           string
		wantCreated    bool
		wantDeleted    bool
		wantUpdated    bool
	}{
		{
			name: "deleted pull secret",
			fakecli: newFakecli(nil, &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			arocli: baseCluster,
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: corev1.ConditionFalse,
				},
			},
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantCreated: true,
		},
		{
			name: "missing arosvc pull secret",
			fakecli: newFakecli(&corev1.Secret{}, &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: corev1.ConditionFalse,
				},
			},
			arocli:      baseCluster,
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
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
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: corev1.ConditionFalse,
				},
			},
			arocli:      baseCluster,
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
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
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: corev1.ConditionFalse,
				},
			},
			arocli:      baseCluster,
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantUpdated: true,
		},
		{
			name: "wrong secret type",
			fakecli: newFakecli(&corev1.Secret{
				Type: corev1.SecretTypeOpaque,
			}, &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: corev1.ConditionFalse,
				},
			},
			arocli:      baseCluster,
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
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
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: corev1.ConditionFalse,
				},
			},
			arocli: baseCluster,
			want:   `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
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
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: corev1.ConditionTrue,
				},
			},
			arocli: baseCluster,
			want:   `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="},"cloud.redhat.com":{"auth":"ZnJlZDplbnRlcg=="}}}`,
		},
		{
			name: "disabled feature, valid RH key present",
			fakecli: newFakecli(&corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			}, &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: corev1.ConditionTrue,
				},
			},
			arocli: newFakeAro(
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status:     arov1alpha1.ClusterStatus{},
				}),
			want: `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
		},
		{
			name: "disabled feature, valid RH key missing",
			fakecli: newFakecli(&corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			}, &corev1.Secret{Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: corev1.ConditionFalse,
				},
			},
			arocli: newFakeAro(
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status:     arov1alpha1.ClusterStatus{},
				}),
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

			r := &PullSecretReconciler{
				kubernetescli: tt.fakecli,
				log:           logrus.NewEntry(logrus.StandardLogger()),
				arocli:        tt.arocli,
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
				t.Fatalf("Unexpected created state: \ngot: %t\nwant: %t", created, tt.wantCreated)
			}

			if deleted != tt.wantDeleted {
				t.Fatalf("Unexpected deleted state: \ngot: %t\nwant: %t", deleted, tt.wantDeleted)
			}

			if updated != tt.wantUpdated {
				t.Fatalf("Unexpected updated state: \ngot: %t\nwant: %t", updated, tt.wantUpdated)
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

		CONDITIONS:
			for _, condition := range cluster.Status.Conditions {
				for _, wantCondition := range tt.wantConditions {
					if condition.Type == wantCondition.Type && condition.Status == wantCondition.Status {
						continue CONDITIONS
					}
				}
				t.Fatalf("Condition not found in cluster status.\ngot: %v\n want: %v", cluster.Status.Conditions, tt.wantConditions)
			}
		})
	}
}

func TestCheckRHRegistryKeys(t *testing.T) {
	test := []struct {
		name        string
		ps          map[string]string
		wantKeys    []string
		wantMissing string
		wantErr     string
	}{
		{
			name: "without rh key",
			ps: map[string]string{
				"arosvc.azrecr.io": "ZnJlZDplbnRlcg==",
			},
			wantKeys: []string{},
		},
		{
			name: "with rh key",
			ps: map[string]string{
				"arosvc.azrecr.io":   "ZnJlZDplbnRlcg==",
				"registry.redhat.io": "ZnJlZDplbnRlcg==",
			},
			wantKeys: []string{"registry.redhat.io"},
		},
		{
			name: "with multiple rh key",
			ps: map[string]string{
				"arosvc.azrecr.io":   "ZnJlZDplbnRlcg==",
				"registry.redhat.io": "ZnJlZDplbnRlcg==",
				"cloud.redhat.com":   "ZnJlZDplbnRlcg==",
			},
			wantKeys: []string{"registry.redhat.io", "cloud.redhat.com"},
		},
		{
			name: "with one rh key missing",
			ps: map[string]string{
				"arosvc.azrecr.io":   "ZnJlZDplbnRlcg==",
				"registry.redhat.io": "ZnJlZDplbnRlcg==",
			},
			wantKeys:    []string{"registry.redhat.io", "cloud.redhat.com"},
			wantMissing: "cloud.redhat.com",
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			r := &PullSecretReconciler{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}

			out := r.checkRHRegistryKey(tt.ps)
		KEYS:
			for _, wantKey := range tt.wantKeys {
				for _, outKey := range out {
					if outKey == wantKey {
						continue KEYS
					}
				}
				if wantKey == tt.wantMissing {
					// to verify false positives
					continue KEYS
				}
				t.Fatalf("Cannot find key: %s in condition keys", wantKey)
			}
		})
	}
}

func TestKeyCondition(t *testing.T) {
	test := []struct {
		name          string
		failed        bool
		keys          []string
		wantCondition status.Condition
		wantErr       string
	}{
		{
			name:   "cannot parse keys",
			failed: true,
			keys:   []string{},
			wantCondition: status.Condition{
				Type:    arov1alpha1.RedHatKeyPresent,
				Status:  corev1.ConditionFalse,
				Message: "Cannot parse pull-secret",
				Reason:  "CheckFailed",
			},
		},
		{
			name: "no key found",
			keys: []string{},
			wantCondition: status.Condition{
				Type:    arov1alpha1.RedHatKeyPresent,
				Status:  corev1.ConditionFalse,
				Message: "No Red Hat key found in pull-secret",
				Reason:  "CheckDone",
			},
		},
		{
			name: "keys found",
			keys: []string{"registry.redhat.io"},
			wantCondition: status.Condition{
				Type:    arov1alpha1.RedHatKeyPresent,
				Status:  corev1.ConditionTrue,
				Message: "registry.redhat.io,",
				Reason:  "CheckDone",
			},
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			r := &PullSecretReconciler{}

			out := r.keyCondition(tt.failed, tt.keys)
			if !reflect.DeepEqual(out, &tt.wantCondition) {
				t.Fatalf("Condition does not match. want: %v, got: %v", tt.wantCondition, out)
			}
		})
	}
}

func TestBuildRedHatKeyCondition(t *testing.T) {
	test := []struct {
		name          string
		secret        corev1.Secret
		wantCondition status.Condition
	}{
		{
			name: "RH pull secret is present",
			secret: corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantCondition: status.Condition{
				Type:    arov1alpha1.RedHatKeyPresent,
				Status:  corev1.ConditionTrue,
				Reason:  "CheckDone",
				Message: "registry.redhat.io,",
			},
		},
		{
			name: "RH pull secret is missing",
			secret: corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantCondition: status.Condition{
				Type:    arov1alpha1.RedHatKeyPresent,
				Status:  corev1.ConditionFalse,
				Reason:  "CheckDone",
				Message: "No Red Hat key found in pull-secret",
			},
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			r := &PullSecretReconciler{}

			out := r.buildRedHatKeyCondition(&tt.secret)
			if !reflect.DeepEqual(*out, tt.wantCondition) {
				t.Fatalf("Condition does not match: \n%v\n%v", out, tt.wantCondition)
			}
		})
	}

}

func TestFixAndUpdateGlobalPullSecret(t *testing.T) {
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
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
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
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`bad`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			wantError: "invalid character 'b' looking for beginning of value",
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
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`bad`),
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			wantError: "nil operator secret, cannot verify userData integrity",
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			r := &PullSecretReconciler{
				kubernetescli: tt.fakecli,
			}

			err := r.fixAndUpdateGlobalPullSecret(context.Background(), tt.operatorPullSecret, tt.pullSecret)
			if err != nil && (err.Error() != tt.wantError) {
				t.Fatalf("Unexpected error\ngot: %s\nwant: %s", err.Error(), tt.wantError)
			}

			s, err := r.kubernetescli.CoreV1().Secrets("openshift-config").Get(context.Background(), "pull-secret", metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(s, tt.wantSecret) {
				t.Fatalf("Unexpected secret mismatch\ngot: %v\nwant: %v", s, tt.wantSecret)
			}

		})
	}
}
