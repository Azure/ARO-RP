package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureACRToken(t *testing.T) {
	ctx := context.Background()
	clusterUUID := "512a50c8-2a43-4c2a-8fd9-a5539475df2a"

	for _, tt := range []struct {
		name    string
		oc      func() *api.OpenShiftCluster
		wantErr string
	}{
		{
			name: "not found",
			oc: func() *api.OpenShiftCluster {
				return &api.OpenShiftCluster{}
			},
			wantErr: `No object found`,
		},
		{
			name: "expired",
			oc: func() *api.OpenShiftCluster {
				return &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						RegistryProfiles: []*api.RegistryProfile{
							{
								Name:     "test",
								Username: "testuser",
								Expiry:   &date.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
							},
						},
					},
				}
			},
			wantErr: `TerminalError: ACR token has expired`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_, log := testlog.New()

			builder := fake.NewClientBuilder()
			ch := clienthelper.NewWithClient(log, testclienthelper.NewHookingClient(builder.Build()))
			tc := testtasks.NewFakeTestContext(
				ctx, _env, log, func() time.Time { return time.Unix(100, 0) },
				testtasks.WithClientHelper(ch),
				testtasks.WithOpenShiftClusterProperties(clusterUUID, tt.oc().Properties),
			)

			err := EnsureACRTokenIsValid(tc)
			if tt.wantErr != "" && err != nil {
				g.Expect(err).To(MatchError(tt.wantErr))
			} else if tt.wantErr != "" && err == nil {
				t.Errorf("wanted error %s", tt.wantErr)
			} else if tt.wantErr == "" {
				g.Expect(err).ToNot(HaveOccurred())
			}
		})
	}
}
