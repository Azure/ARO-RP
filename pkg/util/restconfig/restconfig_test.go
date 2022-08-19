package restconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
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
			apiServerPrivateEndpointIP: "127.0.0.1",
			dialAddress:                "1.1.1.1:6443",
			dialNetwork:                "tcp",
			wantAddress:                ":6443",
		},
		{
			name:                       "invalid address",
			apiServerPrivateEndpointIP: "127.0.0.1",
			dialAddress:                "1.1.1.1",
			dialNetwork:                "tcp",
			wantErr:                    "address 1.1.1.1: missing port in address",
		},
		{
			name:                       "unsupported network",
			apiServerPrivateEndpointIP: "127.0.0.1",
			dialAddress:                "1.1.1.1:6443",
			dialNetwork:                "udp",
			wantErr:                    "unimplemented network \"udp\"",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			testCtx := context.WithValue(context.Background(), struct{ nonEmptyCtx string }{}, 1)

			oc := &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					NetworkProfile: api.NetworkProfile{
						APIServerPrivateEndpointIP: tt.apiServerPrivateEndpointIP,
					},
				},
			}
			dial := DialContext(oc)

			if tt.wantAddress != "" {
				l, err := net.Listen("tcp", tt.wantAddress)
				if err != nil {
					t.Error(err)
				}
				defer l.Close()
			}

			_, err := dial(testCtx, tt.dialNetwork, tt.dialAddress)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
