package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"

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
	registryResourceID = "/subscriptions/93aeba23-2f76-4307-be82-02921df010cf/resourceGroups/global/providers/Microsoft.ContainerRegistry/registries/arointsvc"
	clusterUUID        = "512a50c8-2a43-4c2a-8fd9-a5539475df2a"
	publicACR          = "arosvc.azurecr.io"
	intACR             = "arointsvc.azurecr.io"
	user               = "testuser"
)

func TestEnsureACRToken(t *testing.T) {
	ctx := context.Background()

	startOf20204 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	hundredDaysInThePast := time.Now().UTC().AddDate(0, 0, -100)
	fiftyDaysInThePast := time.Now().UTC().AddDate(0, 0, -50)

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
			name:     "No issue date",
			azureEnv: azureclient.PublicCloud,
			oc: func() *api.OpenShiftCluster {
				return &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						RegistryProfiles: []*api.RegistryProfile{
							{
								Name:      intACR,
								Username:  user,
								IssueDate: nil,
							},
						},
					},
				}
			},
			wantErr: "TerminalError: no issue date detected, please rotate token",
		},
		{
			name:     "Expired",
			azureEnv: azureclient.PublicCloud,
			oc: func() *api.OpenShiftCluster {
				return &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						RegistryProfiles: []*api.RegistryProfile{
							{
								Name:      publicACR,
								Username:  user,
								IssueDate: &startOf20204,
							},
							{
								Name:      intACR,
								Username:  user,
								IssueDate: &hundredDaysInThePast,
							},
						},
					},
				}
			},
			wantErr: "TerminalError: azure container registry (acr) token is not valid, 100 days have passed",
		},
		{
			name:     "Should rotate token",
			azureEnv: azureclient.PublicCloud,
			oc: func() *api.OpenShiftCluster {
				return &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						RegistryProfiles: []*api.RegistryProfile{
							{
								Name:      publicACR,
								Username:  user,
								IssueDate: &startOf20204,
							},
							{
								Name:      intACR,
								Username:  user,
								IssueDate: &fiftyDaysInThePast,
							},
						},
					},
				}
			},
			wantErr: "TerminalError: 50 days have passed since azure container registry (acr) token was issued, please rotate the token now",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().ACRResourceID().AnyTimes().Return(registryResourceID)
			_env.EXPECT().ACRDomain().AnyTimes().Return(intACR)
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
