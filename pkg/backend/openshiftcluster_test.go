package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/backend/openshiftcluster"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_openshiftcluster "github.com/Azure/ARO-RP/pkg/util/mocks/openshiftcluster"
	testdb "github.com/Azure/ARO-RP/test/database"
	testlog "github.com/Azure/ARO-RP/test/util/log"
	teststatsd "github.com/Azure/ARO-RP/test/util/statsd"
)

type backendTestStruct struct {
	name                            string
	clusterDoc                      *api.OpenShiftClusterDocument
	subscriptionDoc                 *api.SubscriptionDocument
	mocks                           func(*mock_openshiftcluster.MockManagerInterface, *database.Database)
	expectedLogs                    []testlog.ExpectedLogEntry
	expectedDocumentExists          bool
	expectedProvisioningState       api.ProvisioningState
	expectedFailedProvisioningState api.ProvisioningState
	expectedAdminUpdateError        string
	expectedInstallPhase            api.InstallPhase
}

func TestBackendTry(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)

	for _, tt := range []backendTestStruct{
		{
			name: "StateCreating with an InstallPhase set keeps it in Creating",
			clusterDoc: &api.OpenShiftClusterDocument{
				ID:  uuid.NewV4().String(),
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
			},
			subscriptionDoc: &api.SubscriptionDocument{
				ID: mockSubID,
			},
			mocks: func(manager *mock_openshiftcluster.MockManagerInterface, db *database.Database) {
				manager.EXPECT().Create(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
					_, err := db.OpenShiftClusters.Patch(ctx, strings.ToLower(resourceID), func(inFlightDoc *api.OpenShiftClusterDocument) error {
						inFlightDoc.OpenShiftCluster.Properties.Install = &api.Install{}
						return nil
					})
					return err
				})
			},
			expectedDocumentExists:    true,
			expectedProvisioningState: api.ProvisioningStateCreating,
			expectedInstallPhase:      api.InstallPhaseBootstrap,
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					Message: "dequeued",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "creating",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "done",
					Level:   logrus.InfoLevel,
				},
			},
		},
		{
			name: "StateCreating without an InstallPhase marks provisioning as succeeded",
			clusterDoc: &api.OpenShiftClusterDocument{
				ID:  uuid.NewV4().String(),
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
			},
			subscriptionDoc: &api.SubscriptionDocument{
				ID: mockSubID,
			},
			mocks: func(manager *mock_openshiftcluster.MockManagerInterface, db *database.Database) {
				manager.EXPECT().Create(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
					_, err := db.OpenShiftClusters.Patch(ctx, strings.ToLower(resourceID), func(inFlightDoc *api.OpenShiftClusterDocument) error {
						inFlightDoc.OpenShiftCluster.Properties.Install = nil
						return nil
					})
					return err
				})
			},
			expectedDocumentExists:    true,
			expectedProvisioningState: api.ProvisioningStateSucceeded,
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					Message: "dequeued",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "creating",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "done",
					Level:   logrus.InfoLevel,
				},
			},
		},
		{
			name: "StateCreating that fails marks provisioning as Failed",
			clusterDoc: &api.OpenShiftClusterDocument{
				ID:  uuid.NewV4().String(),
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
			},
			subscriptionDoc: &api.SubscriptionDocument{
				ID: mockSubID,
			},
			mocks: func(manager *mock_openshiftcluster.MockManagerInterface, db *database.Database) {
				manager.EXPECT().Create(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
					return errors.New("something bad!")
				})
			},
			expectedDocumentExists:          true,
			expectedProvisioningState:       api.ProvisioningStateFailed,
			expectedFailedProvisioningState: api.ProvisioningStateCreating,
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					Message: "dequeued",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "creating",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "done",
					Level:   logrus.InfoLevel,
				},
			},
		},
		{
			name: "StateAdminUpdating success",
			clusterDoc: &api.OpenShiftClusterDocument{
				ID:  uuid.NewV4().String(),
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:       resourceID,
					Name:     "resourceName",
					Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
					Location: "location",
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState:     api.ProvisioningStateAdminUpdating,
						LastProvisioningState: api.ProvisioningStateSucceeded,
					},
				},
			},
			subscriptionDoc: &api.SubscriptionDocument{
				ID: mockSubID,
			},
			mocks: func(manager *mock_openshiftcluster.MockManagerInterface, db *database.Database) {
				manager.EXPECT().AdminUpdate(gomock.Any()).Return(nil)
			},
			expectedDocumentExists:    true,
			expectedProvisioningState: api.ProvisioningStateSucceeded,
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					Message: "dequeued",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "admin updating",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "done",
					Level:   logrus.InfoLevel,
				},
			},
		},
		{
			name: "StateAdminUpdating failure",
			clusterDoc: &api.OpenShiftClusterDocument{
				ID:  uuid.NewV4().String(),
				Key: strings.ToLower(resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:       resourceID,
					Name:     "resourceName",
					Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
					Location: "location",
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState:     api.ProvisioningStateAdminUpdating,
						LastProvisioningState: api.ProvisioningStateSucceeded,
					},
				},
			},
			subscriptionDoc: &api.SubscriptionDocument{
				ID: mockSubID,
			},
			mocks: func(manager *mock_openshiftcluster.MockManagerInterface, db *database.Database) {
				manager.EXPECT().AdminUpdate(gomock.Any()).Return(errors.New("oh no!"))
			},
			expectedDocumentExists:          true,
			expectedProvisioningState:       api.ProvisioningStateSucceeded,
			expectedFailedProvisioningState: "",
			expectedAdminUpdateError:        "oh no!",
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					Message: "dequeued",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "admin updating",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "done",
					Level:   logrus.InfoLevel,
				},
			},
		},
		{
			name: "StateDeleting success",
			clusterDoc: &api.OpenShiftClusterDocument{
				ID:  uuid.NewV4().String(),
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
			},
			subscriptionDoc: &api.SubscriptionDocument{
				ID: mockSubID,
			},
			mocks: func(manager *mock_openshiftcluster.MockManagerInterface, db *database.Database) {
				manager.EXPECT().Delete(gomock.Any()).Return(nil)
			},
			expectedDocumentExists: false,
			expectedLogs: []testlog.ExpectedLogEntry{
				{
					Message: "dequeued",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "deleting",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "done",
					Level:   logrus.InfoLevel,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			controller := gomock.NewController(t)
			defer controller.Finish()

			manager := mock_openshiftcluster.NewMockManagerInterface(controller)
			_env := mock_env.NewMockDev(controller)

			h, log := testlog.NewCapturingLogger()

			db, _, err := testdb.NewDatabase(ctx, log)
			if err != nil {
				t.Fatal(err)
			}

			createManager := func(*logrus.Entry, env.Interface, database.OpenShiftClusters, billing.Manager, *api.OpenShiftClusterDocument, *api.SubscriptionDocument) (openshiftcluster.ManagerInterface, error) {
				return manager, nil
			}

			_, err = db.OpenShiftClusters.Create(ctx, tt.clusterDoc)
			if err != nil {
				t.Fatal(err)
			}

			_, err = db.Subscriptions.Create(ctx, tt.subscriptionDoc)
			if err != nil {
				t.Fatal(err)
			}

			tt.mocks(manager, db)

			b, err := newBackend(ctx, log, _env, db, teststatsd.New())
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

			out, err := db.OpenShiftClusters.Get(ctx, strings.ToLower(resourceID))
			if tt.expectedDocumentExists {
				if err != nil {
					t.Fatal(err)
				}

				if out.OpenShiftCluster.Properties.LastAdminUpdateError != tt.expectedAdminUpdateError {
					t.Errorf("LastAdminUpdateError was %s not %s", out.OpenShiftCluster.Properties.LastAdminUpdateError, tt.expectedAdminUpdateError)
				}

				if out.OpenShiftCluster.Properties.ProvisioningState != tt.expectedProvisioningState {
					t.Errorf("ProvisioningState was %s not %s", out.OpenShiftCluster.Properties.ProvisioningState, tt.expectedProvisioningState)
				}

				if tt.expectedProvisioningState == api.ProvisioningStateCreating {
					if out.OpenShiftCluster.Properties.Install == nil {
						t.Error("install phase was nil")
					} else {
						if out.OpenShiftCluster.Properties.Install.Phase != tt.expectedInstallPhase {
							t.Errorf("InstallPhase was %s not %s", out.OpenShiftCluster.Properties.Install.Phase, tt.expectedInstallPhase)
						}
					}
				} else if tt.expectedProvisioningState == api.ProvisioningStateFailed {
					if out.OpenShiftCluster.Properties.FailedProvisioningState != tt.expectedFailedProvisioningState {
						t.Errorf("FailedProvisioningState was %s not %s", out.OpenShiftCluster.Properties.FailedProvisioningState, tt.expectedFailedProvisioningState)

					}
				}
			} else {
				// We should not have a document to look at
				if out != nil {
					t.Error("got unexpected document")
					t.Error(out)
				}
			}

			errs := testlog.AssertLoggingOutput(h, tt.expectedLogs)
			for _, e := range errs {
				t.Error(e)
			}
		})
	}
}
