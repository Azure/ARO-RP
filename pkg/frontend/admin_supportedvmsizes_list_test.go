package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"testing"

	"github.com/go-test/deep"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api/util/vms"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestSupportedvmsizes(t *testing.T) {
	mastervmsizes := vms.SupportedVMSizesByRole[vms.VMRoleMaster]
	workervmsizes := vms.SupportedVMSizesByRole[vms.VMRoleWorker]

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

func TestSupportedvmsizesIncludesTestingSizesInCI(t *testing.T) {
	controller := gomock.NewController(t)
	mockEnv := mock_env.NewMockInterface(controller)
	mockEnv.EXPECT().IsCI().Return(true)

	f := &frontend{env: mockEnv}
	gotResponse, err := f.supportedVMSizesForRole(vms.VMRoleMaster)
	utilerror.AssertErrorMessage(t, err, "")

	got := map[vms.VMSize]vms.VMSizeStruct{}
	if err := json.Unmarshal(gotResponse, &got); err != nil {
		t.Fatal(err)
	}

	if _, ok := got[vms.VMSizeStandardD4sV3]; !ok {
		t.Fatalf("expected CI master list to include %q, got %v", vms.VMSizeStandardD4sV3, got)
	}
}
