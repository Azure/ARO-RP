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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

var (
	clusterEtcHostsControllerDisabled = &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.EtcHostsEnabled:     operator.FlagFalse,
				operator.ForceReconciliation: operator.FlagTrue,
			},
			Domain:                   "test.com",
			GatewayDomains:           []string{"testgateway.com"},
			APIIntIP:                 "10.10.10.10",
			GatewayPrivateEndpointIP: "20.20.20.20",
		},
	}
	clusterEtcHostsControllerEnabled = &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.EtcHostsEnabled:     operator.FlagTrue,
				operator.EtcHostsManaged:     operator.FlagTrue,
				operator.ForceReconciliation: operator.FlagTrue,
			},
			Domain:                   "test.com",
			GatewayDomains:           []string{"testgateway.com"},
			APIIntIP:                 "10.10.10.10",
			GatewayPrivateEndpointIP: "20.20.20.20",
		},
	}
	clusterEtcHostsControllerEnabledManagedFalse = &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.EtcHostsEnabled:     operator.FlagTrue,
				operator.EtcHostsManaged:     operator.FlagFalse,
				operator.ForceReconciliation: operator.FlagTrue,
			},
			Domain:                   "test.com",
			GatewayDomains:           []string{"testgateway.com"},
			APIIntIP:                 "10.10.10.10",
			GatewayPrivateEndpointIP: "20.20.20.20",
		},
	}
	clusterEtcHostsControllerEnabledReconcileFalse = &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.EtcHostsEnabled:     operator.FlagTrue,
				operator.EtcHostsManaged:     operator.FlagTrue,
				operator.ForceReconciliation: operator.FlagFalse,
			},
			Domain:                   "test.com",
			GatewayDomains:           []string{"testgateway.com"},
			APIIntIP:                 "10.10.10.10",
			GatewayPrivateEndpointIP: "20.20.20.20",
		},
	}
	clusterEtcHostsControllerEnabledManagedFalseReconcileFalse = &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.EtcHostsEnabled:     operator.FlagTrue,
				operator.EtcHostsManaged:     operator.FlagFalse,
				operator.ForceReconciliation: operator.FlagFalse,
			},
			Domain:                   "test.com",
			GatewayDomains:           []string{"testgateway.com"},
			APIIntIP:                 "10.10.10.10",
			GatewayPrivateEndpointIP: "20.20.20.20",
		},
	}
	machinePoolMaster = &mcv1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{Name: "master"},
		Status:     mcv1.MachineConfigPoolStatus{},
		Spec:       mcv1.MachineConfigPoolSpec{},
	}
	machinePoolWorker = &mcv1.MachineConfigPool{
		ObjectMeta: metav1.ObjectMeta{Name: "worker"},
		Status:     mcv1.MachineConfigPoolStatus{},
		Spec:       mcv1.MachineConfigPoolSpec{},
	}
	clusterVersionNotUpdating = &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{
				{
					State:   configv1.CompletedUpdate,
					Version: "4.10.11",
				},
			},
		},
	}
	clusterVersionUpdating = &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Spec: configv1.ClusterVersionSpec{},
		Status: configv1.ClusterVersionStatus{
			Conditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionTrue,
				},
			},
		},
	}
)

func TestReconcileEtcHostsCluster(t *testing.T) {
	type test struct {
		name        string
		objects     []client.Object
		mocks       func(mdh *mock_dynamichelper.MockInterface)
		expectedLog *logrus.Entry
		wantRequeue bool
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
		},
		{
			name: "etchosts controller enabled, managed true, mc exist",
			objects: []client.Object{
				clusterEtcHostsControllerEnabled, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "running"},
			wantRequeue: false,
		},
		{
			name: "etchosts controller enabled, managed true, only master mc exist",
			objects: []client.Object{
				clusterEtcHostsControllerEnabled, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "99-worker-aro-etc-hosts-gateway-domains not found, creating it"},
			wantRequeue: true,
		},
		{
			name: "etchosts controller enabled, managed true, only worker mc exist",
			objects: []client.Object{
				clusterEtcHostsControllerEnabled, machinePoolMaster, machinePoolWorker, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "99-master-aro-etc-hosts-gateway-domains not found, creating it"},
			wantRequeue: true,
		},
		{
			name: "etchosts controller enabled, managed true, no mc exist",
			objects: []client.Object{
				clusterEtcHostsControllerEnabled, machinePoolMaster, machinePoolWorker,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "99-master-aro-etc-hosts-gateway-domains not found, creating it"},
			wantRequeue: true,
		},
		{
			name: "etchosts controller enabled, managed false, reconcile false, cluster not updating, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledManagedFalseReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "running"},
			wantRequeue: false,
		},
		{
			name: "etchosts controller enabled, managed false, reconcile false, cluster updating, mc removed",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledManagedFalseReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "etchosts managed is false, machine configs removed"},
			wantRequeue: false,
		},
		{
			name: "etchosts controller enabled, managed true, reconcile false, cluster not updating, mc exist, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "running"},
			wantRequeue: false,
		},
		{
			name: "etchosts controller enabled, managed true, reconcile false, cluster updating, mc exist, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "running"},
			wantRequeue: false,
		},
		{
			name: "etchosts controller enabled, managed true, reconcile false, cluster not updating, only master mc exist, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "99-worker-aro-etc-hosts-gateway-domains not found, creating it"},
			wantRequeue: true,
		},
		{
			name: "etchosts controller enabled, managed true, reconcile false, cluster updating, only master mc exist, ensure worker mc",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker, etchostsMasterMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "99-worker-aro-etc-hosts-gateway-domains not found, creating it"},
			wantRequeue: true,
		},
		{
			name: "etchosts controller enabled, managed true, reconcile false, cluster not updating, only worker mc exist, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "99-master-aro-etc-hosts-gateway-domains not found, creating it"},
			wantRequeue: true,
		},
		{
			name: "etchosts controller enabled, managed true, reconcile false, cluster updating, only worker mc exist, ensure master mc",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker, etchostsWorkerMCMetadata,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "99-master-aro-etc-hosts-gateway-domains not found, creating it"},
			wantRequeue: true,
		},
		{
			name: "etchosts controller enabled, managed true, reconcile false, cluster not updating, no mc exist, no action",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledReconcileFalse, clusterVersionNotUpdating, machinePoolMaster, machinePoolWorker,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(0)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "99-master-aro-etc-hosts-gateway-domains not found, creating it"},
			wantRequeue: true,
		},
		{
			name: "etchosts controller enabled, managed true, reconcile false, cluster updating, no mc exist, ensure master mc",
			objects: []client.Object{
				clusterEtcHostsControllerEnabledReconcileFalse, clusterVersionUpdating, machinePoolMaster, machinePoolWorker,
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			expectedLog: &logrus.Entry{Level: logrus.DebugLevel, Message: "99-master-aro-etc-hosts-gateway-domains not found, creating it"},
			wantRequeue: true,
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

		r := &EtcHostsClusterReconciler{
			AROController: base.AROController{
				Log:    logrus.NewEntry(logger),
				Client: clientBuilder.Build(),
				Name:   ControllerName,
			},
			dh: mdh,
		}

		request := ctrl.Request{}
		request.Name = "cluster"

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
