package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/operator"
)

type operatorFlagsMergeStrategyStruct struct {
	OperatorFlagsMergeStrategy string
	Cluster                    *api.OpenShiftCluster
}

const (
	operatorFlagsMergeStrategyDefault string = "merge"
	operatorFlagsMergeStrategyMerge   string = "merge"
	operatorFlagsMergeStrategyReset   string = "reset"
)

// When a cluster is edited via the PATCH Cluster Geneva Action (aka an Admin Update)
// the flags given are treated according to the provided Update Strategy,
// provided in operatorFlagsMergeStrategy

// merge (default): The provided cluster flags are laid on top of the cluster’s existing flags.
// reset: The provided cluster flags are laid on top of the default cluster flags,
// essentially ‘resetting’ the flags if no new flags are provided.
func OperatorFlagsMergeStrategy(oc *api.OpenShiftCluster, body []byte) error {
	payload := operatorFlagsMergeStrategyStruct{
		OperatorFlagsMergeStrategy: operatorFlagsMergeStrategyDefault,
		Cluster:                    &api.OpenShiftCluster{},
	}

	err := json.Unmarshal(body, &payload)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	// return error if OperatorFlagsMergeStrategy is not merge or reset, default is merge
	if payload.OperatorFlagsMergeStrategy != operatorFlagsMergeStrategyMerge &&
		payload.OperatorFlagsMergeStrategy != operatorFlagsMergeStrategyReset {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "invalid operatorFlagsMergeStrategy '%s', can only be 'merge' or 'reset'", payload.OperatorFlagsMergeStrategy)
	}
	// return if payload is empty
	if payload.Cluster == nil {
		return nil
	}
	properties := &payload.Cluster.Properties
	if properties == nil || properties.OperatorFlags == nil {
		return nil
	}
	// return nil, if OperatorFlagsMergeStrategy is merge and payload has not operatorFlags
	// return operatorFlags of payload, if OperatorFlagsMergeStrategy is merge and payload has operatorFlags
	// return defaultOperatorFlags, if OperatorFlagsMergeStrategy is reset and payload has not operatorFlags
	// return defaultOperatorFlags + operatorFlags of payload, if OperatorFlagsMergeStrategy is reset and payload has operatorFlags
	if payload.OperatorFlagsMergeStrategy == operatorFlagsMergeStrategyReset {
		oc.Properties.OperatorFlags = operator.DefaultOperatorFlags()
		for operatorflag, value := range payload.Cluster.Properties.OperatorFlags {
			oc.Properties.OperatorFlags[operatorflag] = value
		}
	}

	return nil
}
