package admin

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func TestOperatorFlagsMergeStrategy(t *testing.T) {
	tests := []struct {
		name    string
		oc      *api.OpenShiftCluster
		wantOc  *api.OpenShiftCluster
		body    []byte
		wantErr string
	}{
		{
			name:    "invalid_json",
			oc:      nil,
			body:    []byte(`{{}`),
			wantErr: `400: InvalidRequestContent: : The request content was invalid and could not be deserialized: "invalid character '{' looking for beginning of object key string".`,
		},
		{
			name:    "OperatorFlagsMergeStrategy_is_not_merge_or_reset",
			oc:      nil,
			body:    []byte(`{"operatorFlagsMergeStrategy": "xyz"}`),
			wantErr: `400: InvalidParameter: : invalid operatorFlagsMergeStrategy 'xyz', can only be 'merge' or 'reset'`,
		},
		{
			name: "OperatorFlagsMergeStrategy_payload_is_empty",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					OperatorFlags: api.OperatorFlags{"aro.feature1.enabled": "false"},
				},
			},
			body:    []byte(`{"operatorflagsmergestrategy":"merge"}`),
			wantErr: "",
		},
		{
			name: "OperatorFlagsMergeStrategy_reset",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					OperatorFlags: api.OperatorFlags{"aro.feature1.enabled": "false"},
				},
			},
			body:    []byte(`{"operatorflagsmergestrategy":"reset"}`),
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := OperatorFlagsMergeStrategy(tt.oc, tt.body)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
