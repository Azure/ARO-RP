package restconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_proxy "github.com/Azure/ARO-RP/pkg/util/mocks/proxy"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestDialContext(t *testing.T) {
	for _, tt := range []struct {
		name                       string
		apiServerPrivateEndpointIP string
		dialNetwork                string
		dialAddress                string
		wantAddress                string
		wantErr                    string
	}{
		{
			name:                       "replace host ip",
			apiServerPrivateEndpointIP: "10.0.4.6",
			dialAddress:                "1.1.1.1:6443",
			dialNetwork:                "tcp",
			wantAddress:                "10.0.4.6:6443",
		},
		{
			name:                       "invalid address",
			apiServerPrivateEndpointIP: "10.0.4.6",
			dialAddress:                "1.1.1.1",
			dialNetwork:                "tcp",
			wantErr:                    "address 1.1.1.1: missing port in address",
		},
		{
			name:                       "unsupported network",
			apiServerPrivateEndpointIP: "10.0.4.6",
			dialAddress:                "1.1.1.1:6443",
			dialNetwork:                "udp",
			wantErr:                    "unimplemented network \"udp\"",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			testCtx := context.WithValue(context.Background(), struct{ nonEmptyCtx string }{}, 1)

			dialer := mock_proxy.NewMockDialer(controller)
			if tt.wantAddress != "" {
				dialer.EXPECT().DialContext(testCtx, tt.dialNetwork, tt.wantAddress)
			}

			oc := &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					NetworkProfile: api.NetworkProfile{
						APIServerPrivateEndpointIP: tt.apiServerPrivateEndpointIP,
					},
				},
			}
			dial := DialContext(dialer, oc)

			_, err := dial(testCtx, tt.dialNetwork, tt.dialAddress)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
