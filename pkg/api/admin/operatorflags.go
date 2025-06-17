package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
)

const (
	OperatorFlagsMergeStrategyMerge string = "merge"
	OperatorFlagsMergeStrategyReset string = "reset"
)

// When a cluster is edited via the PATCH Cluster Geneva Action (aka an Admin Update)
// the flags given are treated according to the provided Update Strategy,
// provided in operatorFlagsMergeStrategy

// merge (default): The provided cluster flags are laid on top of the cluster’s existing flags.
// reset: The provided cluster flags are laid on top of the default cluster flags,
// essentially ‘resetting’ the flags if no new flags are provided.
func OperatorFlagsMergeStrategy(oc *api.OpenShiftCluster, body []byte, defaultFlags api.OperatorFlags) error {
	payload := &OpenShiftCluster{}

	err := json.Unmarshal(body, &payload)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", fmt.Sprintf("The request content was invalid and could not be deserialized: %q.", err))
	}

	// if it's empty, use the default of merge, which is performed by
	// deserialising the body JSON later
	if payload.OperatorFlagsMergeStrategy == "" {
		return nil
	}

	// return error if OperatorFlagsMergeStrategy is not merge or reset, default is merge
	if payload.OperatorFlagsMergeStrategy != OperatorFlagsMergeStrategyMerge &&
		payload.OperatorFlagsMergeStrategy != OperatorFlagsMergeStrategyReset {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", fmt.Sprintf("invalid operatorFlagsMergeStrategy '%s', can only be 'merge' or 'reset'", payload.OperatorFlagsMergeStrategy))
	}

	// return nil, if OperatorFlagsMergeStrategy is merge and payload has not operatorFlags
	// return operatorFlags of payload, if OperatorFlagsMergeStrategy is merge and payload has operatorFlags
	// return defaultOperatorFlags, if OperatorFlagsMergeStrategy is reset and payload has not operatorFlags
	// return defaultOperatorFlags + operatorFlags of payload, if OperatorFlagsMergeStrategy is reset and payload has operatorFlags
	if payload.OperatorFlagsMergeStrategy == OperatorFlagsMergeStrategyReset {
		oc.Properties.OperatorFlags = defaultFlags
	}

	return nil
}
