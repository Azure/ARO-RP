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
	clusterUUID = "512a50c8-2a43-4c2a-8fd9-a5539475df2a"
	publicACR   = "arosvc.azurecr.io"
	intACR      = "arointsvc.azurecr.io"
	user        = "testuser"
)

func TestEnsureACRToken(t *testing.T) {
	ctx := context.Background()

	startOf2024 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	expiredTime := startOf2024.AddDate(0, 0, -365)
	duringRenewalTime := startOf2024.AddDate(0, 0, -150)

	for _, tt := range []struct {
		name       string
		azureEnv   azureclient.AROEnvironment
		oc         func() *api.OpenShiftCluster
		wantErr    string
		wantResult string
	}{
		{
			name:     "not found",
			azureEnv: azureclient.PublicCloud,
			oc: func() *api.OpenShiftCluster {
				return &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{},
				}
			},
			wantErr: "TerminalError: no issue date detected, please rotate token",
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
								IssueDate: &expiredTime,
							},
							{
								Name:      intACR,
								Username:  user,
								IssueDate: &expiredTime,
							},
						},
					},
				}
			},
			wantErr: "TerminalError: token is expired",
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
								IssueDate: &startOf2024,
							},
							{
								Name:      intACR,
								Username:  user,
								IssueDate: &duringRenewalTime,
							},
						},
					},
				}
			},
			wantErr: "TerminalError: token able to be renewed for 240h0m0s, 720h0m0s validity remaining, please rotate",
		},
		{
			name:     "valid token",
			azureEnv: azureclient.PublicCloud,
			oc: func() *api.OpenShiftCluster {
				return &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						RegistryProfiles: []*api.RegistryProfile{
							{
								Name:      publicACR,
								Username:  user,
								IssueDate: &startOf2024,
							},
							{
								Name:      intACR,
								Username:  user,
								IssueDate: &startOf2024,
							},
						},
					},
				}
			},
			wantResult: "token validity has 4320h0m0s remaining, should be rotated in 3360h0m0s",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			controller := gomock.NewController(t)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().ACRDomain().AnyTimes().Return(intACR)
			_env.EXPECT().Environment().AnyTimes().Return(&tt.azureEnv)
			_env.EXPECT().Now().AnyTimes().DoAndReturn(func() time.Time {
				return startOf2024
			})
			_, log := testlog.New()

			builder := fake.NewClientBuilder()
			ch := clienthelper.NewWithClient(log, testclienthelper.NewHookingClient(builder.Build()))
			tc := testtasks.NewFakeTestContext(
				ctx, _env, log, func() time.Time { return startOf2024 },
				testtasks.WithClientHelper(ch),
				testtasks.WithOpenShiftClusterDocument(&api.OpenShiftClusterDocument{ID: clusterUUID, OpenShiftCluster: tt.oc()}),
			)

			err := EnsureACRTokenIsValid(tc)
			if tt.wantErr != "" && err != nil {
				utilerror.AssertErrorMessage(t, err, tt.wantErr)
			} else if tt.wantErr != "" && err == nil {
				t.Errorf("wanted error %s, got %v", tt.wantErr, err)
			} else if tt.wantErr == "" {
				g.Expect(err).ToNot(HaveOccurred())
			}

			if tt.wantResult != "" {
				g.Expect(tc.GetResultMessage()).To(Equal(tt.wantResult))
			}
		})
	}
}
