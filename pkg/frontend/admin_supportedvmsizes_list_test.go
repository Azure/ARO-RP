package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"testing"

	"github.com/go-test/deep"

	"github.com/Azure/ARO-RP/pkg/api/util/vms"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestSupportedvmsizes(t *testing.T) {
	mastervmsizes := validate.SupportedVMSizesByRole(vms.VMRoleMaster)
	workervmsizes := validate.SupportedVMSizesByRole(vms.VMRoleWorker)

	type test struct {
		name         string
		vmRole       vms.VMRole
		wantResponse map[vms.VMSize]vms.VMSizeStruct
		wantError    string
	}

	for _, tt := range []*test{
		{
			name:         "vmRole is invalid",
			vmRole:       vms.VMRole("invalidVMRole"),
			wantError:    `400: InvalidParameter: : The provided vmRole 'invalidVMRole' is invalid. vmRole can only be master or worker`,
			wantResponse: nil,
		},
		{
			name:         "vmRole is empty",
			vmRole:       vms.VMRole(""),
			wantError:    `400: InvalidParameter: : The provided vmRole '' is invalid. vmRole can only be master or worker`,
			wantResponse: nil,
		},
		{
			name:         "master as vmRole",
			vmRole:       vms.VMRoleMaster,
			wantError:    "",
			wantResponse: mastervmsizes,
		},
		{
			name:         "worker as vmRole",
			vmRole:       vms.VMRoleWorker,
			wantError:    "",
			wantResponse: workervmsizes,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := &frontend{}
			gotResponse, err := f.supportedVMSizesForRole(tt.vmRole)
			utilerror.AssertErrorMessage(t, err, tt.wantError)
			if gotResponse != nil || tt.wantResponse != nil {
				v := map[vms.VMSize]vms.VMSizeStruct{}
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
