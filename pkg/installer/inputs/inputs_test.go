package inputs

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

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
					BoundServiceAccountSigningKey: pointerutils.ToPtr(api.SecureString("signing-key")),
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
