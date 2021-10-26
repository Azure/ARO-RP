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
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	mock_cluster "github.com/Azure/ARO-RP/pkg/util/mocks/cluster"
	testdatabase "github.com/Azure/ARO-RP/test/database"
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
			name: "StateAdminUpdating success sets the last ProvisioningState and clears LastAdminUpdateError and MaintenanceTask",
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
				manager.EXPECT().AdminUpdate(gomock.Any()).Return(nil)
			},
		},
		{
			name: "StateAdminUpdating run failure populates LastAdminUpdateError and restores previous provisioning state + failed provisioning state",
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

			controller := gomock.NewController(t)
			defer controller.Finish()
			manager := mock_cluster.NewMockInterface(controller)

			dbOpenShiftClusters, clientOpenShiftClusters := testdatabase.NewFakeOpenShiftClusters()
			dbSubscriptions, _ := testdatabase.NewFakeSubscriptions()

			f := testdatabase.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters).WithSubscriptions(dbSubscriptions)
			tt.mocks(manager, dbOpenShiftClusters)
			tt.fixture(f)
			err := f.Create()
			if err != nil {
				t.Fatal(err)
			}

			createManager := func(context.Context, *logrus.Entry, env.Interface, database.OpenShiftClusters, database.Gateway, encryption.AEAD, billing.Manager, *api.OpenShiftClusterDocument, *api.SubscriptionDocument) (cluster.Interface, error) {
				return manager, nil
			}

			b, err := newBackend(ctx, log, nil, nil, nil, nil, dbOpenShiftClusters, dbSubscriptions, nil, &noop.Noop{})
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
