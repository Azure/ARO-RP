package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_deploy "github.com/Azure/ARO-RP/pkg/util/mocks/operator/deploy"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestEnsureAROOperator(t *testing.T) {
	ctx := context.Background()

	const (
		key = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"
	)

	for _, tt := range []struct {
		name    string
		doc     *api.OpenShiftClusterDocument
		mocks   func(*mock_deploy.MockOperator)
		wantErr string
	}{
		{
			name: "create/update success",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						IngressProfiles: []api.IngressProfile{
							{
								Visibility: api.VisibilityPublic,
								Name:       "default",
							},
						},
						ProvisioningState: api.ProvisioningStateAdminUpdating,
						ClusterProfile: api.ClusterProfile{
							Version: "4.8.18",
						},
						NetworkProfile: api.NetworkProfile{
							APIServerPrivateEndpointIP: "1.2.3.4",
						},
					},
				},
			},
			mocks: func(dep *mock_deploy.MockOperator) {
				dep.EXPECT().
					CreateOrUpdate(gomock.Any()).
					Return(nil)
			},
		},
		{
			name: "create/update failure",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						IngressProfiles: []api.IngressProfile{
							{
								Visibility: api.VisibilityPublic,
								Name:       "default",
							},
						},
						ProvisioningState: api.ProvisioningStateAdminUpdating,
						ClusterProfile: api.ClusterProfile{
							Version: "4.8.18",
						},
						NetworkProfile: api.NetworkProfile{
							APIServerPrivateEndpointIP: "1.2.3.4",
						},
					},
				},
			},
			mocks: func(dep *mock_deploy.MockOperator) {
				dep.EXPECT().
					CreateOrUpdate(gomock.Any()).
					Return(errors.New("Mock return: CreateFailed"))
			},

			wantErr: "Mock return: CreateFailed",
		},
		{
			name: "enriched data not available - skip",

			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						IngressProfiles:   nil,
						ProvisioningState: api.ProvisioningStateAdminUpdating,
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			dep := mock_deploy.NewMockOperator(controller)
			if tt.mocks != nil {
				tt.mocks(dep)
			}

			m := &manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: tt.doc,

				aroOperatorDeployer: dep,
			}

			err := m.ensureAROOperator(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestAroDeploymentReady(t *testing.T) {
	ctx := context.Background()

	const (
		key = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"
	)

	for _, tt := range []struct {
		name    string
		doc     *api.OpenShiftClusterDocument
		mocks   func(*mock_deploy.MockOperator)
		wantRes bool
	}{
		{
			name: "operator is ready",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						IngressProfiles: []api.IngressProfile{
							{
								Visibility: api.VisibilityPublic,
								Name:       "default",
							},
						},
					},
				},
			},
			mocks: func(dep *mock_deploy.MockOperator) {
				dep.EXPECT().
					IsReady(gomock.Any()).
					Return(true, nil)
			},
			wantRes: true,
		},
		{
			name: "operator is not ready",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						IngressProfiles: []api.IngressProfile{
							{
								Visibility: api.VisibilityPublic,
								Name:       "default",
							},
						},
					},
				},
			},
			mocks: func(dep *mock_deploy.MockOperator) {
				dep.EXPECT().
					IsReady(gomock.Any()).
					Return(false, nil)
			},
			wantRes: false,
		},
		{
			name: "enriched data not available - skip",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						IngressProfiles: nil,
					},
				},
			},
			wantRes: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			dep := mock_deploy.NewMockOperator(controller)
			if tt.mocks != nil {
				tt.mocks(dep)
			}

			m := &manager{
				log:                 logrus.NewEntry(logrus.StandardLogger()),
				doc:                 tt.doc,
				aroOperatorDeployer: dep,
			}

			ok, _, err := m.aroDeploymentReady(ctx)
			if err != nil || ok != tt.wantRes {
				t.Error(err)
			}
		})
	}
}

func TestEnsureAROOperatorRunningDesiredVersion(t *testing.T) {
	ctx := context.Background()

	const (
		key = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"
	)

	for _, tt := range []struct {
		name    string
		doc     *api.OpenShiftClusterDocument
		mocks   func(*mock_deploy.MockOperator)
		wantRes bool
	}{
		{
			name: "operator is runningDesiredVersion",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						IngressProfiles: []api.IngressProfile{
							{
								Visibility: api.VisibilityPublic,
								Name:       "default",
							},
						},
					},
				},
			},
			mocks: func(dep *mock_deploy.MockOperator) {
				dep.EXPECT().
					IsRunningDesiredVersion(gomock.Any()).
					Return(true, nil)
			},
			wantRes: true,
		},
		{
			name: "operator is not runningDesiredVersion",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						IngressProfiles: []api.IngressProfile{
							{
								Visibility: api.VisibilityPublic,
								Name:       "default",
							},
						},
					},
				},
			},
			mocks: func(dep *mock_deploy.MockOperator) {
				dep.EXPECT().
					IsRunningDesiredVersion(gomock.Any()).
					Return(false, nil)
			},
			wantRes: false,
		},
		{
			name: "enriched data not available - skip",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						IngressProfiles: nil,
					},
				},
			},
			wantRes: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			dep := mock_deploy.NewMockOperator(controller)
			if tt.mocks != nil {
				tt.mocks(dep)
			}

			m := &manager{
				log:                 logrus.NewEntry(logrus.StandardLogger()),
				doc:                 tt.doc,
				aroOperatorDeployer: dep,
			}

			ok, _, err := m.ensureAROOperatorRunningDesiredVersion(ctx)
			if err != nil || ok != tt.wantRes {
				t.Error(err)
			}
		})
	}
}
