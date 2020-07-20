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
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/backend/openshiftcluster"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
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
	mocks                           func(*mock_openshiftcluster.MockManagerInterface, database.OpenShiftClusters)
	fixture                         func(*testdb.Fixture)
	expectedLogs                    []map[string]types.GomegaMatcher
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
			fixture: func(f *testdb.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
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
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
				})
			},
			mocks: func(manager *mock_openshiftcluster.MockManagerInterface, dbOpenShiftClusters database.OpenShiftClusters) {
				manager.EXPECT().Create(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
					_, err := dbOpenShiftClusters.Patch(ctx, strings.ToLower(resourceID), func(inFlightDoc *api.OpenShiftClusterDocument) error {
						inFlightDoc.OpenShiftCluster.Properties.Install = &api.Install{}
						return nil
					})
					return err
				})
			},
			expectedDocumentExists:    true,
			expectedProvisioningState: api.ProvisioningStateCreating,
			expectedInstallPhase:      api.InstallPhaseBootstrap,
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("dequeued"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("creating"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("done"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
		},
		{
			name: "StateCreating without an InstallPhase marks provisioning as succeeded",
			fixture: func(f *testdb.Fixture) {
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
			mocks: func(manager *mock_openshiftcluster.MockManagerInterface, dbOpenShiftClusters database.OpenShiftClusters) {
				manager.EXPECT().Create(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
					_, err := dbOpenShiftClusters.Patch(ctx, strings.ToLower(resourceID), func(inFlightDoc *api.OpenShiftClusterDocument) error {
						inFlightDoc.OpenShiftCluster.Properties.Install = nil
						return nil
					})
					return err
				})
			},
			expectedDocumentExists:    true,
			expectedProvisioningState: api.ProvisioningStateSucceeded,
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("dequeued"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("creating"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("done"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
		},
		{
			name: "StateCreating that fails marks provisioning as Failed",
			fixture: func(f *testdb.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
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
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
				})
			},
			mocks: func(manager *mock_openshiftcluster.MockManagerInterface, dbOpenShiftClusters database.OpenShiftClusters) {
				manager.EXPECT().Create(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
					return errors.New("something bad!")
				})
			},
			expectedDocumentExists:          true,
			expectedProvisioningState:       api.ProvisioningStateFailed,
			expectedFailedProvisioningState: api.ProvisioningStateCreating,
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("dequeued"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("creating"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("done"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
		},
		{
			name: "StateAdminUpdating success",
			fixture: func(f *testdb.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
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
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
				})
			},
			mocks: func(manager *mock_openshiftcluster.MockManagerInterface, dbOpenShiftClusters database.OpenShiftClusters) {
				manager.EXPECT().AdminUpdate(gomock.Any()).Return(nil)
			},
			expectedDocumentExists:    true,
			expectedProvisioningState: api.ProvisioningStateSucceeded,
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("dequeued"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("admin updating"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("done"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
		},
		{
			name: "StateAdminUpdating failure",
			fixture: func(f *testdb.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
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
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
				})
			},
			mocks: func(manager *mock_openshiftcluster.MockManagerInterface, dbOpenShiftClusters database.OpenShiftClusters) {
				manager.EXPECT().AdminUpdate(gomock.Any()).Return(errors.New("oh no!"))
			},
			expectedDocumentExists:          true,
			expectedProvisioningState:       api.ProvisioningStateSucceeded,
			expectedFailedProvisioningState: "",
			expectedAdminUpdateError:        "oh no!",
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("dequeued"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("admin updating"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("done"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
		},
		{
			name: "StateDeleting success",
			fixture: func(f *testdb.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
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
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
				})
			},
			mocks: func(manager *mock_openshiftcluster.MockManagerInterface, dbOpenShiftClusters database.OpenShiftClusters) {
				manager.EXPECT().Delete(gomock.Any()).Return(nil)
			},
			expectedDocumentExists: false,
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("dequeued"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("deleting"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("done"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			h, log := testlog.New()

			controller := gomock.NewController(t)
			defer controller.Finish()
			manager := mock_openshiftcluster.NewMockManagerInterface(controller)
			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().DeploymentMode().Return(deployment.Development)

			dbOpenShiftClusters, _ := testdb.NewFakeOpenShiftClusters()
			dbSubscriptions, _ := testdb.NewFakeSubscriptions()

			f := testdb.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters).WithSubscriptions(dbSubscriptions)
			tt.mocks(manager, dbOpenShiftClusters)
			tt.fixture(f)
			err := f.Create()
			if err != nil {
				t.Fatal(err)
			}

			createManager := func(*logrus.Entry, env.Interface, database.OpenShiftClusters, encryption.Cipher, billing.Manager, *api.OpenShiftClusterDocument, *api.SubscriptionDocument) (openshiftcluster.ManagerInterface, error) {
				return manager, nil
			}

			b, err := newBackend(ctx, log, _env, nil, nil, dbOpenShiftClusters, dbSubscriptions, nil, teststatsd.New())
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

			out, err := dbOpenShiftClusters.Get(ctx, strings.ToLower(resourceID))
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

			err = testlog.AssertLoggingOutput(h, tt.expectedLogs)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
