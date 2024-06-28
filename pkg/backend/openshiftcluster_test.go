package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/hive"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	mock_cluster "github.com/Azure/ARO-RP/pkg/util/mocks/cluster"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/deterministicuuid"
	testlog "github.com/Azure/ARO-RP/test/util/log"
	"github.com/Azure/ARO-RP/test/util/testliveconfig"
)

type backendTestStruct struct {
	name    string
	mocks   func(*mock_cluster.MockInterface, database.OpenShiftClusters)
	fixture func(*testdatabase.Fixture)
	checker func(*testdatabase.Checker)
}

func TestBackendTry(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)

	for _, tt := range []backendTestStruct{
		{
			name: "StateCreating success that sets an InstallPhase stays it in Creating",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       resourceID,
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
						Location: "location",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateCreating,
						},
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
				})
			},
			checker: func(c *testdatabase.Checker) {
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       resourceID,
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
						Location: "location",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateCreating,
							Install: &api.Install{
								Phase: api.InstallPhaseBootstrap,
							},
						},
					},
				})
			},
			mocks: func(manager *mock_cluster.MockInterface, dbOpenShiftClusters database.OpenShiftClusters) {
				manager.EXPECT().Install(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
					_, err := dbOpenShiftClusters.Patch(ctx, strings.ToLower(resourceID), func(inFlightDoc *api.OpenShiftClusterDocument) error {
						inFlightDoc.OpenShiftCluster.Properties.Install = &api.Install{}
						return nil
					})
					return err
				})
			},
		},
		{
			name: "StateCreating success without an InstallPhase marks provisioning as succeeded",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       resourceID,
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
						Location: "location",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateCreating,
						},
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
				})
			},
			checker: func(c *testdatabase.Checker) {
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       resourceID,
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
						Location: "location",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
						},
					},
				})
			},
			mocks: func(manager *mock_cluster.MockInterface, dbOpenShiftClusters database.OpenShiftClusters) {
				manager.EXPECT().Install(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
					_, err := dbOpenShiftClusters.Patch(ctx, strings.ToLower(resourceID), func(inFlightDoc *api.OpenShiftClusterDocument) error {
						inFlightDoc.OpenShiftCluster.Properties.Install = nil
						return nil
					})
					return err
				})
			},
		},
		{
			name: "StateCreating that fails marks ProvisioningState as Failed",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       resourceID,
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
						Location: "location",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateCreating,
						},
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
				})
			},
			checker: func(c *testdatabase.Checker) {
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:      strings.ToLower(resourceID),
					Dequeues: 1,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       resourceID,
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
						Location: "location",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateCreating,
						},
					},
				})
			},
			mocks: func(manager *mock_cluster.MockInterface, dbOpenShiftClusters database.OpenShiftClusters) {
				manager.EXPECT().Install(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
					return errors.New("something bad!")
				})
			},
		},
		{
			name: "StateAdminUpdating success sets the last ProvisioningState, clears LastAdminUpdateError and MaintenanceTask, and has maintenance state none",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       resourceID,
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
						Location: "location",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							LastAdminUpdateError:  "oh no",
							MaintenanceTask:       api.MaintenanceTaskEverything,
							MaintenanceState:      api.MaintenanceStateUnplanned,
						},
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
				})
			},
			checker: func(c *testdatabase.Checker) {
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       resourceID,
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
						Location: "location",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							MaintenanceState:  api.MaintenanceStateNone,
						},
					},
				})
			},
			mocks: func(manager *mock_cluster.MockInterface, dbOpenShiftClusters database.OpenShiftClusters) {
				manager.EXPECT().AdminUpdate(gomock.Any()).Return(nil)
			},
		},
		{
			name: "StateAdminUpdating run failure populates LastAdminUpdateError, restores previous provisioning state + failed provisioning state, and sets maintenance state to ongoing",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       resourceID,
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
						Location: "location",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateAdminUpdating,
							LastProvisioningState:   api.ProvisioningStateSucceeded,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							MaintenanceTask:         api.MaintenanceTaskEverything,
							MaintenanceState:        api.MaintenanceStateUnplanned,
						},
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
				})
			},
			checker: func(c *testdatabase.Checker) {
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       resourceID,
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
						Location: "location",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateSucceeded,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							LastAdminUpdateError:    "oh no!",
							MaintenanceState:        api.MaintenanceStateUnplanned,
						},
					},
				})
			},
			mocks: func(manager *mock_cluster.MockInterface, dbOpenShiftClusters database.OpenShiftClusters) {
				manager.EXPECT().AdminUpdate(gomock.Any()).Return(errors.New("oh no!"))
			},
		},
		{
			name: "StateDeleting success deletes the document",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       resourceID,
						Name:     "resourceName",
						Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
						Location: "location",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateDeleting,
						},
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
				})
			},
			checker: func(c *testdatabase.Checker) {},
			mocks: func(manager *mock_cluster.MockInterface, dbOpenShiftClusters database.OpenShiftClusters) {
				manager.EXPECT().Delete(gomock.Any()).Return(nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			log := logrus.NewEntry(logrus.StandardLogger())
			tlc := testliveconfig.NewTestLiveConfig(false, false)

			controller := gomock.NewController(t)
			defer controller.Finish()
			manager := mock_cluster.NewMockInterface(controller)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().LiveConfig().AnyTimes().Return(tlc)

			dbOpenShiftClusters, clientOpenShiftClusters := testdatabase.NewFakeOpenShiftClusters()
			dbSubscriptions, _ := testdatabase.NewFakeSubscriptions()
			uuidGen := deterministicuuid.NewTestUUIDGenerator(deterministicuuid.OPENSHIFT_VERSIONS)
			dbOpenShiftVersions, _ := testdatabase.NewFakeOpenShiftVersions(uuidGen)

			f := testdatabase.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters).WithSubscriptions(dbSubscriptions)
			tt.mocks(manager, dbOpenShiftClusters)
			tt.fixture(f)
			err := f.Create()
			if err != nil {
				t.Fatal(err)
			}

			createManager := func(context.Context, *logrus.Entry, env.Interface, database.OpenShiftClusters, database.Gateway, database.OpenShiftVersions, encryption.AEAD, billing.Manager, *api.OpenShiftClusterDocument, *api.SubscriptionDocument, hive.ClusterManager, metrics.Emitter) (cluster.Interface, error) {
				return manager, nil
			}

			b, err := newBackend(ctx, log, _env, nil, nil, nil, dbOpenShiftClusters, dbSubscriptions, dbOpenShiftVersions, nil, &noop.Noop{})
			if err != nil {
				t.Fatal(err)
			}

			b.ocb = &openShiftClusterBackend{
				backend:    b,
				newManager: createManager,
			}

			worked, err := b.ocb.try(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if !worked {
				t.Fatal("didnt do work")
			}

			// wait on the workers to finish their tasks
			b.waitForWorkerCompletion()

			c := testdatabase.NewChecker()
			tt.checker(c)

			errs := c.CheckOpenShiftClusters(clientOpenShiftClusters)
			for _, err := range errs {
				t.Error(err)
			}
		})
	}
}

func TestAsyncOperationResultLog(t *testing.T) {
	for _, tt := range []struct {
		name                     string
		initialProvisioningState api.ProvisioningState
		backendErr               error
		wantEntries              []map[string]types.GomegaMatcher
	}{
		{
			name:                     "Success Status Code",
			initialProvisioningState: api.ProvisioningStateSucceeded,
			backendErr: &api.CloudError{
				StatusCode: http.StatusNoContent,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeResourceNotFound,
					Message: "This is not a real error",
					Target:  "target",
				},
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"LOGKIND":       gomega.Equal("asyncqos"),
					"operationType": gomega.Equal("Succeeded"),
					"resultType":    gomega.Equal(utillog.SuccessResultType),
				},
			},
		},
		{
			name:                     "User Error Status Code",
			initialProvisioningState: api.ProvisioningStateFailed,
			backendErr: &api.CloudError{
				StatusCode: http.StatusBadRequest,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeResourceNotFound,
					Message: "This is a user error result type",
					Target:  "target",
				},
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"LOGKIND":       gomega.Equal("asyncqos"),
					"operationType": gomega.Equal("Failed"),
					"resultType":    gomega.Equal(utillog.UserErrorResultType),
					"errorDetails":  gomega.ContainSubstring("This is a user error result type"),
				}},
		},
		{
			name:                     "Server Error Status Code",
			initialProvisioningState: api.ProvisioningStateFailed,
			backendErr: &api.CloudError{
				StatusCode: http.StatusInternalServerError,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInternalServerError,
					Message: "This is a server error result type",
					Target:  "target",
				},
			},
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"LOGKIND":       gomega.Equal("asyncqos"),
					"operationType": gomega.Equal("Failed"),
					"resultType":    gomega.Equal(utillog.ServerErrorResultType),
					"errorDetails":  gomega.ContainSubstring("This is a server error result type"),
				}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, log := testlog.New()

			ocb := &openShiftClusterBackend{}
			ocb.asyncOperationResultLog(log, tt.initialProvisioningState, tt.backendErr)
			err := testlog.AssertLoggingOutput(h, tt.wantEntries)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
