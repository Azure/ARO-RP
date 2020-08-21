// stringifying representations of API documents for debugging and testing
// logging

package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"
	"testing"
)

func TestSubscriptionDocumentString(t *testing.T) {
	doc := &SubscriptionDocument{ID: "test"}
	stringed := fmt.Sprint(doc)
	if !strings.Contains(stringed, "test") {
		t.Fatalf("SubscriptionDocument did not format: %s", stringed)
	}
}

func TestOpenShiftClusterDocumentString(t *testing.T) {
	doc := &OpenShiftClusterDocument{
		ID: "test",
		OpenShiftCluster: &OpenShiftCluster{
			Properties: OpenShiftClusterProperties{
				KubeadminPassword: SecureString("SECRET"),
			},
		},
	}
	stringed := fmt.Sprint(doc)
	if !strings.Contains(stringed, "test") {
		t.Fatalf("OpenShiftClusterDocument did not format: %s", stringed)
	}

	// no secrets should survive
	if strings.Contains(stringed, "SECRET") {
		t.Fatalf("OpenShiftClusterDocument did not hide secrets: %s", stringed)
	}
	if !strings.Contains(stringed, "[REDACTED]") {
		t.Fatalf("OpenShiftClusterDocument did not hide secrets: %s", stringed)
	}
}

func TestBillingDocumentString(t *testing.T) {
	doc := &BillingDocument{ID: "test"}
	stringed := fmt.Sprint(doc)
	if !strings.Contains(stringed, "test") {
		t.Fatalf("BillingDocument did not format: %s", stringed)
	}
}
