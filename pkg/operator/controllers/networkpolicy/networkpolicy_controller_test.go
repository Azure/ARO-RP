package networkpolicy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	mock_dynamichelper "github.com/Azure/ARO-RP/pkg/util/mocks/dynamichelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestNetworkPolicyReconciler(t *testing.T) {
	transitionTime := metav1.Time{Time: time.Now()}
	defaultAvailable := utilconditions.ControllerDefaultAvailable(ControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(ControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(ControllerName)

	defaultConditions := []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded}

	clusterVersion417 := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{
				{
					State:   configv1.CompletedUpdate,
					Version: "4.17.12",
				},
			},
		},
	}

	clusterVersion416 := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{
				{
					State:   configv1.CompletedUpdate,
					Version: "4.16.20",
				},
			},
		},
	}

	type test struct {
		name           string
		instance       *arov1alpha1.Cluster
		clusterversion *configv1.ClusterVersion
		mocks          func(mdh *mock_dynamichelper.MockInterface)
		wantConditions []operatorv1.OperatorCondition
		wantErr        string
	}

	for _, tt := range []*test{
		{
			name:           "Failure to get instance",
			mocks:          func(mdh *mock_dynamichelper.MockInterface) {},
			wantConditions: defaultConditions,
			wantErr:        `clusters.aro.openshift.io "cluster" not found`,
		},
		{
			name: "Feature flag disabled",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.CCOPprofNetworkPolicyEnabled: operator.FlagFalse,
					},
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: defaultConditions,
				},
			},
			clusterversion: clusterVersion417,
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Times(0)
			},
			wantConditions: defaultConditions,
		},
		{
			name: "Cluster version below 4.17 skips AdminNetworkPolicy",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.CCOPprofNetworkPolicyEnabled: operator.FlagTrue,
					},
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: defaultConditions,
				},
			},
			clusterversion: clusterVersion416,
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Times(0)
			},
			wantConditions: defaultConditions,
		},
		{
			name: "Cluster version 4.17+ ensures AdminNetworkPolicy",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.CCOPprofNetworkPolicyEnabled: operator.FlagTrue,
					},
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: defaultConditions,
				},
			},
			clusterversion: clusterVersion417,
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			wantConditions: defaultConditions,
		},
		{
			name: "Ensure fails sets degraded",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.CCOPprofNetworkPolicyEnabled: operator.FlagTrue,
					},
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: defaultConditions,
				},
			},
			clusterversion: clusterVersion417,
			mocks: func(mdh *mock_dynamichelper.MockInterface) {
				mdh.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(errors.New("failed to ensure"))
			},
			wantErr: "failed to ensure",
			wantConditions: []operatorv1.OperatorCondition{
				defaultAvailable,
				defaultProgressing,
				{
					Type:               ControllerName + "Controller" + operatorv1.OperatorStatusTypeDegraded,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: transitionTime,
					Message:            "failed to ensure",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mdh := mock_dynamichelper.NewMockInterface(controller)
			tt.mocks(mdh)

			clientBuilder := testclienthelper.NewAROFakeClientBuilder()
			if tt.instance != nil {
				clientBuilder = clientBuilder.WithObjects(tt.instance)
			}
			if tt.clusterversion != nil {
				clientBuilder = clientBuilder.WithObjects(tt.clusterversion)
			}

			ctx := context.Background()

			r := NewReconciler(
				logrus.NewEntry(logrus.StandardLogger()),
				clientBuilder.Build(),
				mdh,
			)

			request := ctrl.Request{}
			request.Name = "cluster"

			_, err := r.Reconcile(ctx, request)

			if tt.instance != nil {
				utilconditions.AssertControllerConditions(t, ctx, r.Client, tt.wantConditions)
			}

			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
