package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/status"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroFake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

func TestPullSecretReconciler(t *testing.T) {
	newFakecli := func(s *v1.Secret, c *v1.Secret) *fake.Clientset {
		c.ObjectMeta = metav1.ObjectMeta{
			Name:      operator.SecretName,
			Namespace: operator.Namespace,
		}
		c.Type = v1.SecretTypeOpaque
		if s == nil {
			return fake.NewSimpleClientset(c)
		}

		s.ObjectMeta = metav1.ObjectMeta{
			Name:      "pull-secret",
			Namespace: "openshift-config",
		}
		if s.Type == "" {
			s.Type = v1.SecretTypeDockerConfigJson
		}
		return fake.NewSimpleClientset(s, c)
	}

	newFakeAro := func(a *arov1alpha1.Cluster) *aroFake.Clientset {
		return aroFake.NewSimpleClientset(a)
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
		arocli         *aroFake.Clientset
		wantConditions []status.Condition
		wantErr        bool
		want           string
		wantCreated    bool
		wantDeleted    bool
		wantUpdated    bool
	}{
		{
			name: "deleted pull secret",
			fakecli: newFakecli(nil, &v1.Secret{Data: map[string][]byte{
				v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			arocli: baseCluster,
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: v1.ConditionFalse,
				},
			},
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantCreated: true,
		},
		{
			name: "missing arosvc pull secret",
			fakecli: newFakecli(&v1.Secret{}, &v1.Secret{Data: map[string][]byte{
				v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: v1.ConditionFalse,
				},
			},
			arocli:      baseCluster,
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantUpdated: true,
		},
		{
			name: "modified arosvc pull secret",
			fakecli: newFakecli(&v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":""}}}`),
				},
			}, &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				}}),
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: v1.ConditionFalse,
				},
			},
			arocli:      baseCluster,
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantUpdated: true,
		},
		{
			name: "unparseable secret",
			fakecli: newFakecli(&v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`bad`),
				},
			}, &v1.Secret{Data: map[string][]byte{
				v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: v1.ConditionFalse,
				},
			},
			arocli:      baseCluster,
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantUpdated: true,
		},
		{
			name: "wrong secret type",
			fakecli: newFakecli(&v1.Secret{
				Type: v1.SecretTypeOpaque,
			}, &v1.Secret{Data: map[string][]byte{
				v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: v1.ConditionFalse,
				},
			},
			arocli:      baseCluster,
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
			}, &v1.Secret{Data: map[string][]byte{
				v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: v1.ConditionFalse,
				},
			},
			arocli: baseCluster,
			want:   `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
		},
		{
			name: "valid RH key present",
			fakecli: newFakecli(&v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			}, &v1.Secret{Data: map[string][]byte{
				v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: v1.ConditionTrue,
				},
			},
			arocli: baseCluster,
			want:   `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
		},
		{
			name: "disabled feature, valid RH key present",
			fakecli: newFakecli(&v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			}, &v1.Secret{Data: map[string][]byte{
				v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: v1.ConditionTrue,
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
			fakecli: newFakecli(&v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			}, &v1.Secret{Data: map[string][]byte{
				v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			wantConditions: []status.Condition{
				{
					Type:   arov1alpha1.RedHatKeyPresent,
					Status: v1.ConditionFalse,
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

			if s.Type != v1.SecretTypeDockerConfigJson {
				t.Errorf("Unexpected secret type: %s", s.Type)
			}

			if string(s.Data[v1.DockerConfigJsonKey]) != tt.want {
				t.Fatalf("Unexpected secret data.\ngot: %s\nwant: %s", string(s.Data[v1.DockerConfigJsonKey]), tt.want)
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

func TestParseRegistryKeys(t *testing.T) {
	test := []struct {
		name     string
		ps       *v1.Secret
		wantAuth pullsecret.SerializedAuthMap
		wantErr  string
	}{
		{
			name: "ok secret",
			ps: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}, "registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantAuth: pullsecret.SerializedAuthMap{Auths: map[string]pullsecret.SerializedAuth{
				"arosvc.azurecr.io":  {Auth: "ZnJlZDplbnRlcg=="},
				"registry.redhat.io": {Auth: "ZnJlZDplbnRlcg=="},
			}},
		},
		{
			name: "broken secret",
			ps: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantErr: "invalid character ':' after object key:value pair",
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			out, err := pullsecret.UnmarshalSecretData(tt.ps)
			if err != nil {
				if err.Error() != tt.wantErr {
					t.Fatal(err.Error())
				}
			} else if !reflect.DeepEqual(*out, tt.wantAuth) {
				t.Fatalf("Auth does not match:\n%v\n%v", *out, tt.wantAuth)
			}
		})
	}
}

func TestCheckRHRegistryKeys(t *testing.T) {
	test := []struct {
		name     string
		ps       pullsecret.SerializedAuthMap
		wantKeys bool
		wantErr  string
	}{
		{
			name: "without rh key",
			ps: pullsecret.SerializedAuthMap{Auths: map[string]pullsecret.SerializedAuth{
				"arosvc.azurecr.io": {Auth: "ZnJlZDplbnRlcg=="},
			}},
			wantKeys: false,
		},
		{
			name: "with rh key",
			ps: pullsecret.SerializedAuthMap{Auths: map[string]pullsecret.SerializedAuth{
				"arosvc.azurecr.io":  {Auth: "ZnJlZDplbnRlcg=="},
				"registry.redhat.io": {Auth: "ZnJlZDplbnRlcg=="},
			}},
			wantKeys: true,
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			r := &PullSecretReconciler{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}

			out := r.checkRHRegistryKeys(&tt.ps)
			if out != tt.wantKeys {
				t.Fatal("Cannot match keys")
			}
		})
	}
}

func TestKeyCondition(t *testing.T) {
	test := []struct {
		name          string
		failed        bool
		keys          bool
		wantCondition status.Condition
		wantErr       string
	}{
		{
			name:   "cannot parse keys",
			failed: true,
			keys:   false,
			wantCondition: status.Condition{
				Type:    arov1alpha1.RedHatKeyPresent,
				Status:  v1.ConditionFalse,
				Message: "Cannot parse pull-secret",
				Reason:  "CheckFailed",
			},
		},
		{
			name: "no key found",
			keys: false,
			wantCondition: status.Condition{
				Type:    arov1alpha1.RedHatKeyPresent,
				Status:  v1.ConditionFalse,
				Message: "No Red Hat key found in pull-secret",
				Reason:  "CheckDone",
			},
		},
		{
			name: "keys found",
			keys: true,
			wantCondition: status.Condition{
				Type:    arov1alpha1.RedHatKeyPresent,
				Status:  v1.ConditionTrue,
				Message: "Red Hat registry key present in pull-secret",
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

func TestUpdateRedHatKeyCondition(t *testing.T) {
	test := []struct {
		name          string
		secret        v1.Secret
		wantCondition status.Condition
	}{
		{
			name: "RH pull secret is present",
			secret: v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantCondition: status.Condition{
				Type:    arov1alpha1.RedHatKeyPresent,
				Status:  v1.ConditionTrue,
				Reason:  "CheckDone",
				Message: "Red Hat registry key present in pull-secret",
			},
		},
		{
			name: "RH pull secret is missing",
			secret: v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantCondition: status.Condition{
				Type:    arov1alpha1.RedHatKeyPresent,
				Status:  v1.ConditionFalse,
				Reason:  "CheckDone",
				Message: "No Red Hat key found in pull-secret",
			},
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			r := &PullSecretReconciler{}

			out, err := r.updateRedHatKeyCondition(&tt.secret)
			if err != nil {
				t.Fatalf("Unexpected error")
			}
			if !reflect.DeepEqual(*out, tt.wantCondition) {
				t.Fatalf("Condition does not match: \n%v\n%v", out, tt.wantCondition)
			}
		})
	}

}

func TestUpdateGlobalPullSecret(t *testing.T) {
	test := []struct {
		name               string
		operatorPullSecret *v1.Secret
		pullSecret         *v1.Secret
		wantSecret         *v1.Secret
		wantAction         PullSecretAction
		wantError          string
	}{
		{
			name: "Red Hat Key present",
			pullSecret: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			operatorPullSecret: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantSecret: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			wantAction: NoAction,
			wantError:  "",
		},
		{
			name: "Red Hat Key missing",
			pullSecret: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			operatorPullSecret: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantSecret: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			wantAction: UpdatePullSecret,
		},
		{
			name:       "Secret empty",
			pullSecret: &v1.Secret{},
			operatorPullSecret: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			wantSecret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			wantAction: RecreatePullSecret,
		},
		{
			name:       "Secret missing",
			pullSecret: nil,
			operatorPullSecret: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			wantSecret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			wantAction: CreatePullSecret,
		},
		{
			name: "Red Hat Key present but secret type broken",
			pullSecret: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: v1.SecretTypeBasicAuth,
			},
			operatorPullSecret: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantSecret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="},"registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			wantAction: RecreatePullSecret,
			wantError:  "",
		},
		{
			name: "Secret auth key broken broken",
			pullSecret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"lbnRlcg=="}}}`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			operatorPullSecret: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantSecret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			wantAction: UpdatePullSecret,
		},
		{
			name: "Secret not parseable",
			pullSecret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`bad`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			operatorPullSecret: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantSecret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			wantAction: UpdatePullSecret,
		},
		{
			name: "Operator secret not parseable",
			pullSecret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`bad`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			operatorPullSecret: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`bad`),
				},
			},
			wantSecret: nil,
			wantAction: NoAction,
			wantError:  "Cannot parse operatorSecret, cannot verify userSecret integrity",
		},
		{
			name: "Operator secret nil",
			pullSecret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`bad`),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
			operatorPullSecret: nil,
			wantSecret:         nil,
			wantAction:         NoAction,
			wantError:          "Nil operator secret, cannot verify userData integrity",
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			r := &PullSecretReconciler{}

			secret, action, err := r.updateGlobalPullSecret(tt.operatorPullSecret, tt.pullSecret)
			if err != nil && err.Error() != tt.wantError {
				t.Errorf("Unexpected error\ngot: %s\nwant: %s", err.Error(), tt.wantError)
			}
			if !reflect.DeepEqual(secret, tt.wantSecret) {
				t.Fatalf("Secrets does not match:\ngot: %v\nwant: %v", secret, tt.wantSecret)
			}
			if action != tt.wantAction {
				t.Fatalf("Unexpected action\ngot: %v\nwant: %v", action, tt.wantAction)
			}
		})
	}
}
