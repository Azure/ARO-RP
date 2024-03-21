package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	operatorv1 "github.com/openshift/api/operator/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestMachineConfigPoolReconciler(t *testing.T) {
	transitionTime := metav1.Time{Time: time.Now()}
	defaultAvailable := utilconditions.ControllerDefaultAvailable(MachineConfigPoolControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(MachineConfigPoolControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(MachineConfigPoolControllerName)
	defaultConditions := []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded}

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
							operator.DnsmasqEnabled: operator.FlagFalse,
						},
					},
				},
			},
			mocks:      func(mdh *mock_dynamichelper.MockInterface) {},
			request:    ctrl.Request{},
			wantErrMsg: "",
		},
		{
			name: "no MachineConfigPool does nothing",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status: arov1alpha1.ClusterStatus{
						Conditions: []operatorv1.OperatorCondition{
							defaultAvailable,
							defaultProgressing,
							{
								Type:               MachineConfigPoolControllerName + "Controller" + operatorv1.OperatorStatusTypeDegraded,
								Status:             operatorv1.ConditionTrue,
								LastTransitionTime: transitionTime,
							},
						},
					},
					Spec: arov1alpha1.ClusterSpec{
						OperatorFlags: arov1alpha1.OperatorFlags{
							operator.DnsmasqEnabled: operator.FlagTrue,
						},
					},
				},
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {},
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "",
					Name:      "custom",
				},
			},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "MachineConfigPool reconciles ARO DNS MachineConfig",
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
					ObjectMeta: metav1.ObjectMeta{
						Name:       "custom",
						Finalizers: []string{MachineConfigPoolControllerName},
					},
					Status: mcv1.MachineConfigPoolStatus{},
					Spec:   mcv1.MachineConfigPoolSpec{},
				},
			},
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.AssignableToTypeOf(&mcv1.MachineConfig{})).Times(1)
			},
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "",
					Name:      "custom",
				},
			},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			createTally := make(map[string]int)
			updateTally := make(map[string]int)

			client := testclienthelper.NewHookingClient(ctrlfake.NewClientBuilder().
				WithObjects(tt.objects...).
				Build())

			client.WithCreateHook(testclienthelper.TallyCountsAndKey(createTally)).WithUpdateHook(testclienthelper.TallyCountsAndKey(updateTally))

			log := logrus.NewEntry(logrus.StandardLogger())
			ch := clienthelper.NewWithClient(log, client)

			r := NewMachineConfigPoolReconciler(
				log,
				client,
				ch,
			)
			ctx := context.Background()
			_, err := r.Reconcile(ctx, tt.request)

			utilerror.AssertErrorMessage(t, err, tt.wantErrMsg)
			utilconditions.AssertControllerConditions(t, ctx, client, tt.wantConditions)
		})
	}
}
