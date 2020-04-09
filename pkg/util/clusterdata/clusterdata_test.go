package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_clusterdata "github.com/Azure/ARO-RP/pkg/util/mocks/clusterdata"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	"github.com/Azure/ARO-RP/test/util/cmp"
)

func TestBestEffortEnricher(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
	defaultMockCtx := context.Background()
	defaultMockTaskConstructors := []enricherTaskConstructor{
		func(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster) (enricherTask, error) {
			return mockEnricherTaskFunc(func(callbacks chan<- func(), errs chan<- error) {
				callbacks <- func() { oc.ID = "changed-id" }
			}), nil
		},
		func(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster) (enricherTask, error) {
			return mockEnricherTaskFunc(func(callbacks chan<- func(), errs chan<- error) {
				callbacks <- func() { oc.Name = "changed-name" }
			}), nil
		},
	}
	defaultMockRestConfig := func(env.Interface, *api.OpenShiftCluster) (*rest.Config, error) {
		return &rest.Config{}, nil
	}

	for _, tt := range []struct {
		name             string
		taskConstructors []enricherTaskConstructor
		restConfig       func(env.Interface, *api.OpenShiftCluster) (*rest.Config, error)
		ctx              func() (context.Context, context.CancelFunc)
		ocs              func() []*api.OpenShiftCluster
		wantOcs          []*api.OpenShiftCluster
	}{
		{
			name: "all changes applied - no error",
			ocs: func() []*api.OpenShiftCluster {
				return []*api.OpenShiftCluster{
					{ID: "old-id-1", Name: "old-name-1"},
					{ID: "old-id-2", Name: "old-name-2"},
				}
			},
			wantOcs: []*api.OpenShiftCluster{
				{ID: "changed-id", Name: "changed-name"},
				{ID: "changed-id", Name: "changed-name"},
			},
		},
		{
			name: "partial changes - error from one of the task constructor",
			taskConstructors: []enricherTaskConstructor{
				defaultMockTaskConstructors[0],
				func(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster) (enricherTask, error) {
					return nil, errors.New("fake error from the task constructor")
				},
			},
			ocs: func() []*api.OpenShiftCluster {
				return []*api.OpenShiftCluster{{ID: "old-id-1"}}
			},
			wantOcs: []*api.OpenShiftCluster{{ID: "changed-id"}},
		},
		{
			name: "partial changes - context cancelled",
			taskConstructors: []enricherTaskConstructor{
				defaultMockTaskConstructors[0],
				func(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster) (enricherTask, error) {
					return mockEnricherTaskFunc(func(callbacks chan<- func(), errs chan<- error) {
						time.Sleep(time.Minute)
						callbacks <- func() { oc.Name = "changed-name" }
					}), nil
				},
			},
			ctx: func() (context.Context, context.CancelFunc) {
				// This is a potential flake. Increase the timeout if needed,
				// but keep it as short as possible to keep tests fast.
				return context.WithTimeout(context.Background(), time.Second)
			},
			ocs: func() []*api.OpenShiftCluster {
				return []*api.OpenShiftCluster{{ID: "old-id-1", Name: "old-name-1"}}
			},
			wantOcs: []*api.OpenShiftCluster{{ID: "changed-id", Name: "old-name-1"}},
		},
		{
			name: "partial changes - error from one of the tasks",
			taskConstructors: []enricherTaskConstructor{
				defaultMockTaskConstructors[0],
				func(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster) (enricherTask, error) {
					return mockEnricherTaskFunc(func(callbacks chan<- func(), errs chan<- error) {
						errs <- errors.New("fake error")
					}), nil
				},
			},
			ocs: func() []*api.OpenShiftCluster {
				return []*api.OpenShiftCluster{{ID: "old-id-1", Name: "old-name-1"}}
			},
			wantOcs: []*api.OpenShiftCluster{{ID: "changed-id", Name: "old-name-1"}},
		},
		{
			name:             "no changes - ProvisioningStateCreating",
			taskConstructors: defaultMockTaskConstructors,
			ocs: func() []*api.OpenShiftCluster {
				return []*api.OpenShiftCluster{{
					ID: "old-id-1",
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateCreating,
					},
				}}
			},
			wantOcs: []*api.OpenShiftCluster{{
				ID: "old-id-1",
				Properties: api.OpenShiftClusterProperties{
					ProvisioningState: api.ProvisioningStateCreating,
				},
			}},
		},
		{
			name:             "no changes - ProvisioningStateDeleting",
			taskConstructors: defaultMockTaskConstructors,
			ocs: func() []*api.OpenShiftCluster {
				return []*api.OpenShiftCluster{{
					ID: "old-id-1",
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateDeleting,
					},
				}}
			},
			wantOcs: []*api.OpenShiftCluster{{
				ID: "old-id-1",
				Properties: api.OpenShiftClusterProperties{
					ProvisioningState: api.ProvisioningStateDeleting,
				},
			}},
		},
		{
			name:             "no changes - Failed ProvisioningStateCreating",
			taskConstructors: defaultMockTaskConstructors,
			ocs: func() []*api.OpenShiftCluster {
				return []*api.OpenShiftCluster{{
					ID: "old-id-1",
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState:       api.ProvisioningStateFailed,
						FailedProvisioningState: api.ProvisioningStateCreating,
					},
				}}
			},
			wantOcs: []*api.OpenShiftCluster{{
				ID: "old-id-1",
				Properties: api.OpenShiftClusterProperties{
					ProvisioningState:       api.ProvisioningStateFailed,
					FailedProvisioningState: api.ProvisioningStateCreating,
				},
			}},
		},
		{
			name:             "no changes - Failed ProvisioningStateDeleting",
			taskConstructors: defaultMockTaskConstructors,
			ocs: func() []*api.OpenShiftCluster {
				return []*api.OpenShiftCluster{{
					ID: "old-id-1",
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState:       api.ProvisioningStateFailed,
						FailedProvisioningState: api.ProvisioningStateDeleting,
					},
				}}
			},
			wantOcs: []*api.OpenShiftCluster{{
				ID: "old-id-1",
				Properties: api.OpenShiftClusterProperties{
					ProvisioningState:       api.ProvisioningStateFailed,
					FailedProvisioningState: api.ProvisioningStateDeleting,
				},
			}},
		},
		{
			name:             "no changes - error loading the rest config",
			taskConstructors: defaultMockTaskConstructors,
			restConfig: func(env.Interface, *api.OpenShiftCluster) (*rest.Config, error) {
				return nil, errors.New("fake error from rest config")
			},
			ocs: func() []*api.OpenShiftCluster {
				return []*api.OpenShiftCluster{{ID: "old-id-1"}}
			},
			wantOcs: []*api.OpenShiftCluster{{ID: "old-id-1"}},
		},
		{
			name:             "no changes on empty list of clusters",
			taskConstructors: defaultMockTaskConstructors,
			restConfig:       defaultMockRestConfig,
			ocs: func() []*api.OpenShiftCluster {
				return nil
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			restConfig := defaultMockRestConfig
			if tt.restConfig != nil {
				restConfig = tt.restConfig
			}

			taskConstructors := defaultMockTaskConstructors
			if tt.taskConstructors != nil {
				taskConstructors = tt.taskConstructors
			}

			e := &bestEffortEnricher{
				log:              log,
				env:              &env.Test{},
				restConfig:       restConfig,
				taskConstructors: taskConstructors,
				m:                &noop.Noop{},
			}

			ctx := defaultMockCtx
			ctxCancel := context.CancelFunc(func() {}) // no op
			if tt.ctx != nil {
				ctx, ctxCancel = tt.ctx()
			}
			defer ctxCancel()

			ocs := tt.ocs()
			docs := make([]*api.OpenShiftClusterDocument, len(ocs), len(ocs))
			for i, oc := range ocs {
				docs[i] = &api.OpenShiftClusterDocument{OpenShiftCluster: oc}
			}
			e.Enrich(ctx, docs...)

			if !reflect.DeepEqual(ocs, tt.wantOcs) {
				t.Error(cmp.Diff(ocs, tt.wantOcs))
			}
		})
	}
}

type mockEnricherTaskFunc func(callbacks chan<- func(), errs chan<- error)

func (m mockEnricherTaskFunc) FetchData(callbacks chan<- func(), errs chan<- error) {
	m(callbacks, errs)
}
func (m mockEnricherTaskFunc) SetDefaults() {}

func TestCachingEnricher(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	cacheTTL := 5 * time.Minute
	mockCurrentTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	mockCacheMissTime := mockCurrentTime.Add(-cacheTTL - 1)
	mockCacheHitTime := mockCurrentTime.Add(cacheTTL - 1)

	testCases := []struct {
		name                  string
		docs                  func() []*api.OpenShiftClusterDocument
		mockOpenshiftClusters func(openshiftClusters *mock_database.MockOpenShiftClusters)
		mockInnerEnricher     func(innerEnricher *mock_clusterdata.MockOpenShiftClusterEnricher)
		wantDocs              []*api.OpenShiftClusterDocument
	}{
		{
			name: "all docs hit cache",
			docs: func() []*api.OpenShiftClusterDocument {
				return []*api.OpenShiftClusterDocument{
					{LastEnrichment: &mockCacheHitTime},
				}
			},
			wantDocs: []*api.OpenShiftClusterDocument{
				{LastEnrichment: &mockCacheHitTime},
			},
		},
		{
			name: "all docs miss cache",
			docs: func() []*api.OpenShiftClusterDocument {
				return []*api.OpenShiftClusterDocument{
					{LastEnrichment: &mockCacheMissTime},
				}
			},
			mockInnerEnricher: func(innerEnricher *mock_clusterdata.MockOpenShiftClusterEnricher) {
				innerEnricher.EXPECT().Enrich(gomock.Any(), []*api.OpenShiftClusterDocument{
					{LastEnrichment: &mockCacheMissTime},
				})
			},
			mockOpenshiftClusters: func(openshiftClusters *mock_database.MockOpenShiftClusters) {
				openshiftClusters.EXPECT().Update(gomock.Any(), &api.OpenShiftClusterDocument{LastEnrichment: &mockCurrentTime})
			},
			wantDocs: []*api.OpenShiftClusterDocument{
				{LastEnrichment: &mockCurrentTime},
			},
		},
		{
			name: "some of the docs miss cache",
			docs: func() []*api.OpenShiftClusterDocument {
				return []*api.OpenShiftClusterDocument{
					{ID: "fake-1", LastEnrichment: &mockCacheHitTime},
					{ID: "fake-2", LastEnrichment: &mockCacheMissTime},
				}
			},
			mockInnerEnricher: func(innerEnricher *mock_clusterdata.MockOpenShiftClusterEnricher) {
				innerEnricher.EXPECT().Enrich(gomock.Any(), []*api.OpenShiftClusterDocument{
					{ID: "fake-2", LastEnrichment: &mockCacheMissTime},
				})
			},
			mockOpenshiftClusters: func(openshiftClusters *mock_database.MockOpenShiftClusters) {
				openshiftClusters.EXPECT().Update(gomock.Any(), &api.OpenShiftClusterDocument{ID: "fake-2", LastEnrichment: &mockCurrentTime})
			},
			wantDocs: []*api.OpenShiftClusterDocument{
				{ID: "fake-1", LastEnrichment: &mockCacheHitTime},
				{ID: "fake-2", LastEnrichment: &mockCurrentTime},
			},
		},
		{
			name: "no docs",
			docs: func() []*api.OpenShiftClusterDocument {
				return nil
			},
		},
	}

	for _, variant := range []struct {
		name                 string
		callEnrichAndPersist bool
	}{
		{
			name: "Enrich",
		},
		{
			name:                 "EnrichAndPersist",
			callEnrichAndPersist: true,
		},
	} {
		t.Run(variant.name, func(t *testing.T) {
			for _, tt := range testCases {
				t.Run(tt.name, func(t *testing.T) {
					controller := gomock.NewController(t)
					defer controller.Finish()

					openshiftClusters := mock_database.NewMockOpenShiftClusters(controller)
					innerEnricher := mock_clusterdata.NewMockOpenShiftClusterEnricher(controller)

					if tt.mockInnerEnricher != nil {
						tt.mockInnerEnricher(innerEnricher)
					}

					if tt.mockOpenshiftClusters != nil && variant.callEnrichAndPersist {
						tt.mockOpenshiftClusters(openshiftClusters)
					}

					e := &cachingEnricher{
						now:      func() time.Time { return mockCurrentTime },
						log:      log,
						db:       &database.Database{OpenShiftClusters: openshiftClusters},
						m:        &noop.Noop{},
						inner:    innerEnricher,
						cacheTTL: cacheTTL,
					}

					docs := tt.docs()
					if variant.callEnrichAndPersist {
						e.EnrichAndPersist(context.Background(), docs...)
					} else {
						e.Enrich(context.Background(), docs...)
					}

					if !reflect.DeepEqual(docs, tt.wantDocs) {
						t.Error(cmp.Diff(docs, tt.wantDocs))
					}
				})
			}
		})
	}

}
