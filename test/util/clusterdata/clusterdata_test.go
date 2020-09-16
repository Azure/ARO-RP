package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestFakeEnricher(t *testing.T) {
	ctx := context.Background()
	rn1 := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"
	rn2 := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName2"

	doc := &api.OpenShiftCluster{
		ID:   rn1,
		Name: "resourceName1",
		Type: "Microsoft.RedHatOpenShift/openshiftClusters",
		Properties: api.OpenShiftClusterProperties{
			ClusterProfile: api.ClusterProfile{
				PullSecret: "{}",
			},
			ServicePrincipalProfile: api.ServicePrincipalProfile{
				ClientSecret: "clientSecret1",
			},
		},
	}

	enricher := NewTestEnricher()

	// No enrichments expected and none done == no errors
	errs := enricher.Check([]string{})
	if len(errs) != 0 {
		t.Error(errs)
	}

	// 1 enrichment done and a matching enrichment done == no errors
	enricher.Enrich(ctx, doc)
	errs = enricher.Check([]string{rn1})
	if len(errs) != 0 {
		t.Error(errs)
	}

	// 1 enrichment done, 1 matching and 1 not matching enrichment done == the
	// missing enrichment is returned in an error
	errs = enricher.Check([]string{rn1, rn2})
	if len(errs) != 1 {
		t.Error(errs)
	} else if !strings.Contains(errs[0].Error(), rn2) {
		t.Errorf("%s did not have %s", errs[0].Error(), rn2)
	}
}
