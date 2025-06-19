package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/api"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
)

func TestInstallConfigMap(t *testing.T) {
	var expected = map[string]string{"install-config.yaml": "apiVersion: v1\nplatform:\n  azure:\n    region: \"testLocation\"\n"}

	r := installConfigCM("testNamespace", "testLocation")

	for _, err := range deep.Equal(r.StringData, expected) {
		t.Error(err)
	}
}

func TestClusterManifestsSecret(t *testing.T) {
	var expected = map[string]string{"custom.yaml": "apiVersion: v1\nkind: Secret\nmetadata:\n  creationTimestamp: null\n  name: demo-credentials\n  namespace: default\nstringData:\n  demo1: value1\n  demo2: value2\ntype: Opaque\n"}
	customManifests := map[string]kruntime.Object{
		"custom.yaml": &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.Identifier(),
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "demo-credentials",
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				"demo1": "value1",
				"demo2": "value2",
			},
		},
	}

	r, _ := clusterManifestsSecret("testNamespace", customManifests)

	for _, err := range deep.Equal(r.StringData, expected) {
		t.Error(err)
	}
	if r.Namespace != "testNamespace" {
		t.Errorf("Incorrect Secret namespace, expected: testNamespace, found %s", r.Namespace)
	}
}

func TestClusterAzureSecret(t *testing.T) {
	for _, tt := range []struct {
		name   string
		oc     *api.OpenShiftCluster
		wantSP bool
	}{
		{
			name: "Successfully return secret - MIWI Cluster",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
				},
			},
			wantSP: false,
		},
		{
			name: "Successfully return secret - Non-MIWI Cluster",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{
						ClientID:     "clientid",
						ClientSecret: api.SecureString("clientsecret"),
					},
				},
			},
			wantSP: true,
		},
		{
			name: "Failed returning secret - Non-MIWI Cluster, No Credentials",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{},
				},
			},
			wantSP: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			subDoc := api.SubscriptionDocument{
				Subscription: &api.Subscription{
					Properties: &api.SubscriptionProperties{
						TenantID: "tenantid",
					},
				},
				ID: "fakeID",
			}
			r, _ := clusterAzureSecret("testNamespace", tt.oc, &subDoc)

			_, ok := r.Data["osServicePrincipal.json"]
			if tt.wantSP != ok {
				t.Errorf("Wanted %t got %t", tt.wantSP, ok)
			}
			if r.Namespace != "testNamespace" {
				t.Errorf("Incorrect Secret namespace, expected: testNamespace, found %s", r.Namespace)
			}
		})
	}
}

func TestBoundSASigningKeySecret(t *testing.T) {
	testNamespace := "aro-UUID"
	for _, tt := range []struct {
		name       string
		oc         *api.OpenShiftCluster
		wantSecret *corev1.Secret
		wantErr    string
	}{
		{
			name: "csp cluster - returns nil",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ServicePrincipalProfile: &api.ServicePrincipalProfile{},
				},
			},
		},
		{
			name: "miwi cluster - boundServiceAccountSigningKey not set - returns error",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					ClusterProfile:                  api.ClusterProfile{},
				},
			},
			wantErr: "properties.clusterProfile.boundServiceAccountSigningKey not set",
		},
		{
			name: "miwi cluster - creates secret with bound-service-account-signing-key.key contents",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					ClusterProfile: api.ClusterProfile{
						BoundServiceAccountSigningKey: to.Ptr(api.SecureString("fakeboundserviceaccountsigningkey")),
					},
				},
			},
			wantSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      boundServiceAccountSigningKeySecretName,
					Namespace: testNamespace,
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					boundServiceAccountSigningKeySecretKey: "fakeboundserviceaccountsigningkey",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			secret, err := boundSASigningKeySecret(testNamespace, tt.oc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			assert.Equal(t, tt.wantSecret, secret)
		})
	}
}
