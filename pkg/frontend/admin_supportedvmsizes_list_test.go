package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"testing"

	"github.com/go-test/deep"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
)

func TestSupportedvmsizes(t *testing.T) {
	mastervmsizes := validate.SupportedVMSizesByRole(validate.VMRoleMaster)
	workervmsizes := validate.SupportedVMSizesByRole(validate.VMRoleWorker)

	type test struct {
		name         string
		vmRole       string
		wantResponse map[api.VMSize]api.VMSizeStruct
		wantError    string
	}

	for _, tt := range []*test{
		{
			name:         "vmRole is invalid",
			vmRole:       "invalidVMRole",
			wantError:    `400: InvalidParameter: : The provided vmRole 'invalidVMRole' is invalid. vmRole can only be master or worker`,
			wantResponse: nil,
		},
		{
			name:         "vmRole is empty",
			vmRole:       "",
			wantError:    `400: InvalidParameter: : The provided vmRole '' is invalid. vmRole can only be master or worker`,
			wantResponse: nil,
		},
		{
			name:         "master as vmRole",
			vmRole:       "master",
			wantError:    "",
			wantResponse: mastervmsizes,
		},
		{
			name:         "worker as vmRole",
			vmRole:       "worker",
			wantError:    "",
			wantResponse: workervmsizes,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := &frontend{}
			gotResponse, err := f.supportedVMSizesForRole(tt.vmRole)
			// err is not nil, but not the expected error
			if err != nil && err.Error() != tt.wantError {
				t.Error(err)
			}
			// err is nil but we expected an error
			if err == nil && tt.wantError != "" {
				t.Errorf("unexpected error %v , wanted error %s", err, tt.wantError)
			}
			// if gotResponse and wantResponse are nil, skip
			if gotResponse != nil || tt.wantResponse != nil {
				v := map[api.VMSize]api.VMSizeStruct{}
				err = json.Unmarshal(gotResponse, &v)
				if err != nil {
					t.Error(err)
				}
				if diff := deep.Equal(v, tt.wantResponse); diff != nil {
					t.Errorf("unexpected response %s, wanted to match %#v (%s)", string(gotResponse), tt.wantResponse, diff)
				}
			}
		})
	}
}
