package etchosts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func TestReconcileEtcHostsMachineConfig(t *testing.T) {
	type test struct {
		name        string
		objects     []client.Object
		mocks       func(mdh *mock_dynamichelper.MockInterface)
		expectedLog *logrus.Entry
		wantRequeue bool
		requestName string
	}

	for _, tt := range []*test{
		{
			name: "etchosts controller disabled",
			objects: []client.Object{
				clusterEtcHostsControllerDisabled,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "controller is disabled"},
			wantRequeue: false,
			requestName: "cluster",
		},
		{
			name: "etchosts controller enabled, managed false",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledManagedFalse, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "etchosts managed is false, machine configs removed"},
			wantRequeue: false,
			requestName: "cluster",
		},
		{
			name: "etchosts controller enabled, managed true, regex not match",
			objects: []client.Object{
				clusterEtcHostsControllerEnabled, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "running"},
			wantRequeue: false,
			requestName: "cluster",
		},
		{
			name: "etchosts controller enabled, managed true, regex match",
			objects: []client.Object{
				clusterEtcHostsControllerEnabled, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "reconcile object openshift-machine-api/99-master-aro-etc-hosts-gateway-domains"},
			wantRequeue: false,
			requestName: "99-master-aro-etc-hosts-gateway-domains",
		},
		{
			name: "etchosts controller enabled, managed false, cluster not updating, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledManagedFalseReconcileFalse, clusterVersionNotUpdating, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "running"},
			wantRequeue: false,
			requestName: "cluster",
		},
		{
			name: "etchosts controller enabled, managed false, cluster updating, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledManagedFalseReconcileFalse, clusterVersionUpdating, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "etchosts managed is false, machine configs removed"},
			wantRequeue: false,
			requestName: "cluster",
		},
		{
			name: "etchosts controller enabled, managed true, cluster not updating, regex not match, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "running"},
			wantRequeue: false,
			requestName: "cluster",
		},
		{
			name: "etchosts controller enabled, managed true, cluster updating, regex not match, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "running"},
			wantRequeue: false,
			requestName: "cluster",
		},
		{
			name: "etchosts controller enabled, managed true, cluster not updating, regex match, reconcile - no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "reconcile object openshift-machine-api/99-master-aro-etc-hosts-gateway-domains"},
			wantRequeue: false,
			requestName: "99-master-aro-etc-hosts-gateway-domains",
		},
		{
			name: "etchosts controller enabled, managed true, cluster updating, regex match, reconcile - ensure machine config",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "reconcile object openshift-machine-api/99-master-aro-etc-hosts-gateway-domains"},
			wantRequeue: false,
			requestName: "99-master-aro-etc-hosts-gateway-domains",
		},
	} {
		controller := gomock.NewController(t)
		defer controller.Finish()

		mdh := mock_dynamichelper.NewMockInterface(controller)

		tt.mocks(mdh)

		ctx := context.Background()

		logger := &logrus.Logger{
			Out:       io.Discard,
			Formatter: new(logrus.TextFormatter),
			Hooks:     make(logrus.LevelHooks),
			Level:     logrus.TraceLevel,
		}
		var hook = logtest.NewLocal(logger)

		clientBuilder := ctrlfake.NewClientBuilder().WithObjects(tt.objects...).WithStatusSubresource(tt.objects...)

		r := &EtcHostsMachineConfigReconciler{
			AROController: base.AROController{
				Log:    logrus.NewEntry(logger),
				Client: clientBuilder.Build(),
				Name:   ControllerName,
			},
			dh: mdh,
		}

		request := ctrl.Request{}
		request.Name = tt.requestName

		result, err := r.Reconcile(ctx, request)
		if err != nil {
			logger.Log(logrus.ErrorLevel, err)
		}

		if tt.wantRequeue != result.Requeue {
			t.Errorf("Test %v | wanted to requeue %v but was set to %v", tt.name, tt.wantRequeue, result.Requeue)
		}

		actualLog := hook.LastEntry()
		logger.Log(logrus.InfoLevel, actualLog)
		if actualLog == nil {
			assert.Equal(t, tt.expectedLog, actualLog)
		} else {
			assert.Equal(t, tt.expectedLog.Level.String(), actualLog.Level.String())
			assert.Equal(t, tt.expectedLog.Message, actualLog.Message)
		}
	}
}
