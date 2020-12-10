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

	baseCluster := newFakeAro(&arov1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster"}, Status: arov1alpha1.ClusterStatus{}})

	tests := []struct {
		name        string
		request     ctrl.Request
		fakecli     *fake.Clientset
		arocli      *aroFake.Clientset
		wantErr     bool
		want        string
		wantCreated bool
		wantDeleted bool
		wantUpdated bool
	}{
		{
			name: "deleted pull secret",
			fakecli: newFakecli(nil, &v1.Secret{Data: map[string][]byte{
				v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
			arocli:      baseCluster,
			want:        `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
			wantCreated: true,
		},
		{
			name: "missing arosvc pull secret",
			fakecli: newFakecli(&v1.Secret{}, &v1.Secret{Data: map[string][]byte{
				v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
			}}),
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
			arocli: baseCluster,
			want:   `{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}}}`,
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
				arocli:        tt.arocli.AroV1alpha1(),
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

			s, err := r.kubernetescli.CoreV1().Secrets("openshift-config").Get(context.Background(), "pull-secret", metav1.GetOptions{})
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

func TestParseRegistryKeys(t *testing.T) {
	test := []struct {
		name     string
		ps       *v1.Secret
		wantAuth serializedAuthMap
		wantErr  string
	}{
		{
			name: "ok secret",
			ps: &v1.Secret{
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte(`{"auths":{"arosvc.azurecr.io":{"auth":"ZnJlZDplbnRlcg=="}, "registry.redhat.io":{"auth":"ZnJlZDplbnRlcg=="}}}`),
				},
			},
			wantAuth: serializedAuthMap{Auths: map[string]serializedAuth{
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
			r := &PullSecretReconciler{}

			out, err := r.parseRHRegistryKeys(tt.ps)
			if err != nil {
				if err.Error() != tt.wantErr {
					t.Fatal(err.Error())
				}
			} else if !reflect.DeepEqual(*out, tt.wantAuth) {
				t.Fatal("Auth does not match")
			}
		})
	}
}

func TestCheckRHRegistryKeys(t *testing.T) {
	test := []struct {
		name     string
		ps       serializedAuthMap
		wantKeys []string
		wantErr  string
	}{
		{
			name: "without rh key",
			ps: serializedAuthMap{Auths: map[string]serializedAuth{
				"arosvc.azurecr.io": {Auth: "ZnJlZDplbnRlcg=="},
			}},
			wantKeys: []string{},
		},
		{
			name: "with rh key",
			ps: serializedAuthMap{Auths: map[string]serializedAuth{
				"arosvc.azurecr.io":  {Auth: "ZnJlZDplbnRlcg=="},
				"registry.redhat.io": {Auth: "ZnJlZDplbnRlcg=="},
			}},
			wantKeys: []string{"registry.redhat.io"},
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			r := &PullSecretReconciler{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}

			out := r.checkRHRegistryKeys(&tt.ps)
			if !reflect.DeepEqual(out, tt.wantKeys) {
				t.Fatal("Cannot match keys")
			}
		})
	}
}

func TestUpdateRHCondition(t *testing.T) {
	test := []struct {
		name          string
		keys          []string
		wantCondition status.Condition
		wantErr       string
	}{
		{
			name: "no keys found",
			keys: []string{},
			wantCondition: status.Condition{
				Type:    arov1alpha1.RedHatKeyPresent,
				Status:  v1.ConditionFalse,
				Message: "No Red Hat registry keys found in pull-secret.",
				Reason:  "CheckDone",
			},
		},
		{
			name: "keys found",
			keys: []string{"registry.redhat.io"},
			wantCondition: status.Condition{
				Type:    arov1alpha1.RedHatKeyPresent,
				Status:  v1.ConditionTrue,
				Message: "Red Hat registry keys present in pull-secret: registry.redhat.io, ",
				Reason:  "CheckDone",
			},
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			r := &PullSecretReconciler{}

			out := r.updateRHKeysCondition(tt.keys)
			if !reflect.DeepEqual(out, &tt.wantCondition) {
				t.Fatalf("Condition does not match. want: %v, got: %v", tt.wantCondition, out)
			}
		})
	}
}
