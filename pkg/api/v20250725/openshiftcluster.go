package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
)

func value[T any](p *T) (zero T) {
	if p != nil {
		return *p
	}
	return zero
}

// OpenShiftClusterList represents a list of OpenShift clusters.
type OpenShiftClusterList struct {
	// The list of OpenShift clusters.
	OpenShiftClusters []*OpenShiftCluster `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}

// OpenShiftCluster represents an Azure Red Hat OpenShift cluster.
type OpenShiftCluster struct {
	generated.OpenShiftCluster
}

// UnmarshalJSON ensures that PATCH replaces tags when the field is present.
func (oc *OpenShiftCluster) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("unmarshalling type %T: %s", oc, err.Error())
	}
	if _, present := fields["tags"]; present {
		oc.Tags = nil
	}

	return json.Unmarshal(data, &oc.OpenShiftCluster)
}

// UsesWorkloadIdentity checks whether a cluster is a Workload Identity cluster or a Service Principal cluster
func (oc *OpenShiftCluster) UsesWorkloadIdentity() bool {
	return oc.Properties != nil && oc.Properties.PlatformWorkloadIdentityProfile != nil && oc.Properties.ServicePrincipalProfile == nil
}
