package platformworkloadidentity

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

func TestGetPlatformWorkloadIdentityFederatedCredName(t *testing.T) {
	subscriptionId := uuid.DefaultGenerator.Generate()
	resourceGroup := "aro-cluster-rg"
	clusterName := "cluster"
	identityName := "identity"
	saName := "system:serviceaccount:openshift-workload:workload"
	sanitizedsaName := strings.ReplaceAll(saName, ":", "-")
	parts := strings.Split(sanitizedsaName, "-")
	sanitizedsaName = strings.Join(parts[2:], "-")

	clusterResourceId, _ := azure.ParseResourceID(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", subscriptionId, resourceGroup, clusterName))
	identityResourceId, _ := azure.ParseResourceID(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAAssignedIdentities/%s", subscriptionId, resourceGroup, identityName))

	t.Run("generates successfully", func(t *testing.T) {
		GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, identityResourceId, saName)
	})

	t.Run("has expected key as prefix", func(t *testing.T) {
		wantPrefix := fmt.Sprintf("%s_%s", sanitizedsaName, clusterName)
		got := GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, identityResourceId, saName)

		if !strings.HasPrefix(got, wantPrefix) {
			t.Errorf("wanted %s to have prefix %s", got, wantPrefix)
		}
	})

	t.Run("generates a consistent result", func(t *testing.T) {
		prev := GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, identityResourceId, saName)
		for i := 0; i < 5; i++ {
			next := GetPlatformWorkloadIdentityFederatedCredName(clusterResourceId, identityResourceId, saName)

			if prev != next {
				t.Fatalf("got inconsistent results for same parameters: %s vs %s, diff %s", next, prev, cmp.Diff(next, prev))
			}
			prev = next
		}
	})
}
