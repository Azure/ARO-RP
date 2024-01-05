package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestClusterReconciler(t *testing.T) {
	transitionTime := metav1.Time{Time: time.Now()}
	defaultAvailable := utilconditions.ControllerDefaultAvailable(ClusterControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(ClusterControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(ClusterControllerName)
	defaultConditions := []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded}

	fakeDh := func(controller *gomock.Controller) *mock_dynamichelper.MockInterface {
		return mock_dynamichelper.NewMockInterface(controller)
	}

	tests := []struct {
		name           string
		objects        []client.Object
		mocks          func(mdh *mock_dynamichelper.MockInterface)
		request        ctrl.Request
		wantErrMsg     string
		wantConditions []operatorv1.OperatorCondition
	}{
		{
			name:       "no cluster",
			objects:    []client.Object{},
			mocks:      func(mdh *mock_dynamichelper.MockInterface) {},
			request:    ctrl.Request{},
			wantErrMsg: "clusters.aro.openshift.io \"cluster\" not found",
		},
		{
			name: "controller disabled",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status: arov1alpha1.ClusterStatus{
						Conditions: defaultConditions,
					},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							operator.DnsmasqEnabled:      operator.FlagFalse,
							operator.ForceReconciliation: operator.FlagTrue,
						},
					},
				},
			},
			mocks:          func(mdh *mock_dynamichelper.MockInterface) {},
			request:        ctrl.Request{},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "no MachineConfigPools does nothing",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status: arov1alpha1.ClusterStatus{
						Conditions: []operatorv1.OperatorCondition{
							defaultAvailable,
							defaultProgressing,
							{
								Type:               ClusterControllerName + "Controller" + operatorv1.OperatorStatusTypeDegraded,
								Status:             operatorv1.ConditionTrue,
								LastTransitionTime: transitionTime,
							},
						},
					},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							operator.DnsmasqEnabled:      operator.FlagTrue,
							operator.ForceReconciliation: operator.FlagTrue,
						},
					},
				},
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any()).Times(1)
			},
			request:        ctrl.Request{},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "valid MachineConfigPool creates ARO DNS MachineConfig",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status: arov1alpha1.ClusterStatus{
						Conditions: defaultConditions,
					},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							operator.DnsmasqEnabled:      operator.FlagTrue,
							operator.ForceReconciliation: operator.FlagTrue,
						},
					},
				},
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{Name: "master"},
					Status:     mcv1.MachineConfigPoolStatus{},
					Spec:       mcv1.MachineConfigPoolSpec{},
				},
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.AssignableToTypeOf(&mcv1.MachineConfig{})).Times(1)
			},
			request:        ctrl.Request{},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "missing a clusterversion fails",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status: arov1alpha1.ClusterStatus{
						Conditions: defaultConditions,
					},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							operator.DnsmasqEnabled: operator.FlagTrue,
						},
					},
				},
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{Name: "master"},
					Status:     mcv1.MachineConfigPoolStatus{},
					Spec:       mcv1.MachineConfigPoolSpec{},
				},
			},
			mocks:      func(mdh *mock_dynamichelper.MockInterface) {},
			request:    ctrl.Request{},
			wantErrMsg: `clusterversions.config.openshift.io "version" not found`,
			wantConditions: []operatorv1.OperatorCondition{
				defaultAvailable, defaultProgressing, {
					Type:               "DnsmasqClusterControllerDegraded",
					Status:             "True",
					Message:            `clusterversions.config.openshift.io "version" not found`,
					LastTransitionTime: transitionTime,
				},
			},
		},
		{
			name: "valid MachineConfigPool, cluster not updating, not forced, does nothing",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status: arov1alpha1.ClusterStatus{
						Conditions: defaultConditions,
					},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							operator.DnsmasqEnabled: operator.FlagTrue,
						},
					},
				},
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{Name: "master"},
					Status:     mcv1.MachineConfigPoolStatus{},
					Spec:       mcv1.MachineConfigPoolSpec{},
				},
				&configv1.ClusterVersion{
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
				},
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Times(0)
			},
			request:        ctrl.Request{},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "valid MachineConfigPool, while cluster updating, creates ARO DNS MachineConfig",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status: arov1alpha1.ClusterStatus{
						Conditions: defaultConditions,
					},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							operator.DnsmasqEnabled: operator.FlagTrue,
						},
					},
				},
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{Name: "master"},
					Status:     mcv1.MachineConfigPoolStatus{},
					Spec:       mcv1.MachineConfigPoolSpec{},
				},
				&configv1.ClusterVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name: "version",
					},
					Spec: configv1.ClusterVersionSpec{
						DesiredUpdate: &configv1.Update{
							Version: "4.10.12",
						},
					},
					Status: configv1.ClusterVersionStatus{
						History: []configv1.UpdateHistory{
							{
								State:   configv1.CompletedUpdate,
								Version: "4.10.11",
							},
						},
					},
				},
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.AssignableToTypeOf(&mcv1.MachineConfig{})).Times(1)
			},

			request:        ctrl.Request{},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			client := ctrlfake.NewClientBuilder().
				WithObjects(tt.objects...).
				Build()

			dh := fakeDh(controller)
			tt.mocks(dh)

			r := NewClusterReconciler(
				logrus.NewEntry(logrus.StandardLogger()),
				client,
				dh,
			)
			ctx := context.Background()
			_, err := r.Reconcile(ctx, tt.request)

			utilerror.AssertErrorMessage(t, err, tt.wantErrMsg)
			utilconditions.AssertControllerConditions(t, ctx, client, tt.wantConditions)
		})
	}
}
