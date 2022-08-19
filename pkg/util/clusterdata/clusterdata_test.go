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
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
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
	defaultMockRestConfig := func(*api.OpenShiftCluster) (*rest.Config, error) {
		return &rest.Config{}, nil
	}

	for _, tt := range []struct {
		name             string
		taskConstructors []enricherTaskConstructor
		restConfig       func(*api.OpenShiftCluster) (*rest.Config, error)
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
			restConfig: func(*api.OpenShiftCluster) (*rest.Config, error) {
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
			controller := gomock.NewController(t)
			defer controller.Finish()

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
			e.Enrich(ctx, ocs...)

			if !reflect.DeepEqual(ocs, tt.wantOcs) {
				t.Error(cmp.Diff(ocs, tt.wantOcs))
			}
		})
	}
}

type mockEnricherTaskFunc func(callbacks chan<- func(), errs chan<- error)

func (m mockEnricherTaskFunc) FetchData(ctx context.Context, callbacks chan<- func(), errs chan<- error) {
	m(callbacks, errs)
}
func (m mockEnricherTaskFunc) SetDefaults() {}
