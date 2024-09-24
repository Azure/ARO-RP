package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_compute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/compute"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestVMSerialConsole(t *testing.T) {
	type test struct {
		name         string
		mocks        func(*mock_compute.MockVirtualMachinesClient)
		wantResponse []byte
		wantError    string
	}

	for _, tt := range []*test{
		{
			name: "basic coverage",
			mocks: func(vmc *mock_compute.MockVirtualMachinesClient) {
				iothing := bytes.NewBufferString("outputhere")

				vmc.EXPECT().GetSerialConsoleForVM(gomock.Any(), clusterRG, "vm1", gomock.Any()).DoAndReturn(func(ctx context.Context,
					rg string, vmName string, target io.Writer) error {
					_, err := io.Copy(target, iothing)
					return err
				})
			},
			wantResponse: []byte(`outputhere`),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			env.EXPECT().Location().AnyTimes().Return(location)

			vmClient := mock_compute.NewMockVirtualMachinesClient(controller)

			tt.mocks(vmClient)
			log := logrus.NewEntry(logrus.StandardLogger())
			a := azureActions{
				log: log,
				env: env,
				oc: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscription, clusterRG),
						},
					},
				},
				virtualMachines: vmClient,
			}

			ctx := context.Background()

			target := &bytes.Buffer{}
			err := a.VMSerialConsole(ctx, log, "vm1", target)

			utilerror.AssertErrorMessage(t, err, tt.wantError)

			for _, errs := range deep.Equal(target.Bytes(), tt.wantResponse) {
				t.Error(errs)
			}
		})
	}
}
