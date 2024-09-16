package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/date"
	"github.com/golang/mock/gomock"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testtasks "github.com/Azure/ARO-RP/test/mimo/tasks"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const (
	registryName       = "arointsvc"
	registryResourceID = "/subscriptions/93aeba23-2f76-4307-be82-02921df010cf/resourceGroups/global/providers/Microsoft.ContainerRegistry/registries/" + registryName
	clusterUUID        = "512a50c8-2a43-4c2a-8fd9-a5539475df2a"
)

func TestEnsureACRToken(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name     string
		azureEnv azureclient.AROEnvironment
		oc       func() *api.OpenShiftCluster
		wantErr  string
	}{
		{
			name:     "not found",
			azureEnv: azureclient.PublicCloud,
			oc: func() *api.OpenShiftCluster {
				return &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{},
				}
			},
			wantErr: "TerminalError: no registry profile detected",
		},
		{
			name:     "No expiry date",
			azureEnv: azureclient.PublicCloud,
			oc: func() *api.OpenShiftCluster {
				return &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						RegistryProfiles: []*api.RegistryProfile{
							{
								Name:      registryName + ".azurecr.io",
								Username:  "testuser",
								IssueDate: nil,
							},
						},
					},
				}
			},
			wantErr: "TerminalError: no expiry date detected",
		},
		{
			name:     "expired",
			azureEnv: azureclient.PublicCloud,
			oc: func() *api.OpenShiftCluster {
				return &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						RegistryProfiles: []*api.RegistryProfile{
							{
								Name:      "arosvc.azurecr.io",
								Username:  "testuser",
								IssueDate: &date.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
							},
							{
								Name:      "arointsvc.azurecr.io",
								Username:  "testuser",
								IssueDate: &date.Time{Time: time.Date(2024, 1, 9, 0, 0, 0, 0, time.UTC)},
							},
						},
					},
				}
			},
			wantErr: "TerminalError: azure container registry (acr) token has expired",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().ACRResourceID().AnyTimes().Return(registryResourceID)
			_env.EXPECT().Environment().AnyTimes().Return(&tt.azureEnv)
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
				utilerror.AssertErrorMessage(t, err, tt.wantErr)
			} else if tt.wantErr != "" && err == nil {
				t.Errorf("wanted error %s", tt.wantErr)
			} else if tt.wantErr == "" {
				g.Expect(err).ToNot(HaveOccurred())
			}
		})
	}
}
