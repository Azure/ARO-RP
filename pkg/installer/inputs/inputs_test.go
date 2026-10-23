package inputs

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestMarshalCustomManifests(t *testing.T) {
	// Typed objects carry no TypeMeta. Without scheme-aware serialization the
	// manifest has no apiVersion or kind and the installer cannot apply it.
	manifests, err := marshalCustomManifests(map[string]kruntime.Object{
		"99_bound_sa_signing_key.yaml": &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "bound-service-account-signing-key", Namespace: "openshift-config"},
			StringData: map[string]string{"service-account.key": "signing-key"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	got := string(manifests["99_bound_sa_signing_key.yaml"])
	for _, want := range []string{"apiVersion: v1", "kind: Secret"} {
		if !strings.Contains(got, want) {
			t.Errorf("manifest is missing %q:\n%s", want, got)
		}
	}
}

func TestProductionHostingEnvironment(t *testing.T) {
	for _, environment := range []azureclient.AROEnvironment{azureclient.PublicCloud, azureclient.USGovernmentCloud} {
		t.Run(environment.Name, func(t *testing.T) {
			t.Setenv("ARO_INSTALLER_IDENTITY_CLIENT_ID", "installer-hosting-client")
			t.Setenv("AZURE_SUBSCRIPTION_ID", "rp-subscription")
			t.Setenv("AZURE_TENANT_ID", "rp-tenant")
			t.Setenv("LOCATION", "eastus")
			t.Setenv("RESOURCEGROUP", "rp-resource-group")
			env := mock_env.NewMockInterface(gomock.NewController(t))
			env.EXPECT().Environment().Return(&environment)
			builder := NewBuilder(env)
			inputs := &InstallerInputs{EnvironmentVariables: map[string]string{}}
			if err := builder.addProductionEnvironment(inputs); err != nil {
				t.Fatal(err)
			}
			for key, want := range map[string]string{
				"ARO_INSTALLER_IDENTITY_CLIENT_ID": "installer-hosting-client",
				"AZURE_ENVIRONMENT":                environment.Name,
				"ARO_AZURE_SUBSCRIPTION_ID":        "rp-subscription",
				"ARO_AZURE_TENANT_ID":              "rp-tenant",
				"ARO_LOCATION":                     "eastus",
				"ARO_RESOURCEGROUP":                "rp-resource-group",
			} {
				if got := inputs.EnvironmentVariables[key]; got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestGetServicePrincipalJSON(t *testing.T) {
	builder := &Builder{}
	oc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ServicePrincipalProfile: &api.ServicePrincipalProfile{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
		},
	}
	sub := &api.SubscriptionDocument{
		ID: "subscription-id",
		Subscription: &api.Subscription{
			Properties: &api.SubscriptionProperties{
				TenantID: "tenant-id",
			},
		},
	}

	got, err := builder.getServicePrincipalJSON(oc, sub)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"subscriptionId": "subscription-id",
		"tenantId":       "tenant-id",
		"clientId":       "client-id",
		"clientSecret":   "client-secret",
	}
	var decoded map[string]string
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatal(err)
	}
	for key, value := range want {
		if decoded[key] != value {
			t.Errorf("%s = %q, want %q", key, decoded[key], value)
		}
	}
}

func TestGetBoundSASigningKey(t *testing.T) {
	builder := &Builder{}

	t.Run("present", func(t *testing.T) {
		oc := &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					BoundServiceAccountSigningKey: new(api.SecureString("signing-key")),
				},
			},
		}

		got, err := builder.getBoundSASigningKey(oc)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "signing-key" {
			t.Fatalf("got %q, want signing-key", got)
		}
	})

	t.Run("missing", func(t *testing.T) {
		_, err := builder.getBoundSASigningKey(&api.OpenShiftCluster{})
		if err == nil {
			t.Fatal("expected an error")
		}
	})
}
