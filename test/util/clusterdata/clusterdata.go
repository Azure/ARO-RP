package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"sort"

	"github.com/go-test/deep"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/clusterdata"
)

type TestEnricher interface {
	clusterdata.OpenShiftClusterEnricher

	Check([]string) []error
}

var _ clusterdata.OpenShiftClusterEnricher = &testEnricher{}
var _ TestEnricher = &testEnricher{}

type testEnricher struct {
	enrichedIDs []string
}

// NewTestEnricher returns an OpenShiftClusterEnricher which passes through
// documents to be enriched.
func NewTestEnricher() *testEnricher {
	return &testEnricher{enrichedIDs: []string{}}
}

func (e *testEnricher) Enrich(ctx context.Context, ocs ...*api.OpenShiftCluster) {
	for _, o := range ocs {
		e.enrichedIDs = append(e.enrichedIDs, o.ID)
	}
}

func (e *testEnricher) Check(expectedIDs []string) (errs []error) {
	sort.Strings(expectedIDs)
	sort.Strings(e.enrichedIDs)
	if len(e.enrichedIDs) == 0 && len(expectedIDs) == 0 {
		return
	}
	if diff := deep.Equal(e.enrichedIDs, expectedIDs); diff != nil {
		for _, e := range diff {
			errs = append(errs, errors.New(e))
		}
	}
	return
}
