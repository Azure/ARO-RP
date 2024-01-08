package machinehealthcheck

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-test/deep"
	configv1 "github.com/openshift/api/config/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

// Test reconcile function
func TestMachineHealthCheckReconciler(t *testing.T) {
	transitionTime := metav1.Time{Time: time.Now()}
	defaultAvailable := utilconditions.ControllerDefaultAvailable(ControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(ControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(ControllerName)

	defaultConditions := []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded}

	clusterversionDefault := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Status: configv1.ClusterVersionStatus{
			Conditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionFalse,
				},
			},
		},
	}
	clusterversionUpgrading := clusterversionDefault.DeepCopy()
	clusterversionUpgrading.Status.Conditions = []configv1.ClusterOperatorStatusCondition{
		{
			Type:   configv1.OperatorProgressing,
			Status: configv1.ConditionTrue,
		},
	}

	type test struct {
		name              string
		instance          *arov1alpha1.Cluster
		clusterversion    *configv1.ClusterVersion
		expectAnnotations map[string]string
		wantCreates       map[string]int
		wantDeletes       map[string]int
		clientHook        func(*testclienthelper.HookingClient)
		wantConditions    []operatorv1.OperatorCondition
		wantErr           string
		wantRequeueAfter  time.Duration
	}

	for _, tt := range []*test{
		{
			name:           "Failure to get instance",
			wantConditions: defaultConditions,
			wantErr:        `clusters.aro.openshift.io "cluster" not found`,
			wantCreates:    map[string]int{},
			wantDeletes:    map[string]int{},
		},
		{
			name: "Enabled Feature Flag is false",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.MachineHealthCheckEnabled: operator.FlagFalse,
					},
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: defaultConditions,
				},
			},
			wantCreates:    map[string]int{},
			wantDeletes:    map[string]int{},
			wantConditions: defaultConditions,
			wantErr:        "",
		},
		{
			name: "Managed Feature Flag is false: ensure mhc and its alert are deleted",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.MachineHealthCheckEnabled: operator.FlagTrue,
						operator.MachineHealthCheckManaged: operator.FlagFalse,
					},
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: defaultConditions,
				},
			},
			wantCreates: map[string]int{},
			wantDeletes: map[string]int{
				"MachineHealthCheck/openshift-machine-api/aro-machinehealthcheck": 1,
				"PrometheusRule/openshift-machine-api/mhc-remediation-alert":      1,
			},
			wantConditions: defaultConditions,
			wantErr:        "",
		},
		{
			name: "Managed Feature Flag is false: mhc fails to delete, an error is returned",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.MachineHealthCheckEnabled: operator.FlagTrue,
						operator.MachineHealthCheckManaged: operator.FlagFalse,
					},
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: defaultConditions,
				},
			},
			clientHook: func(t *testclienthelper.HookingClient) {
				t.WithDeleteHook(func(obj client.Object) error {
					if client.ObjectKeyFromObject(obj).String() == mhcNamespacedName.String() {
						return errors.New("Could not delete mhc")
					}
					return nil
				})
			},
			wantErr: "Could not delete mhc",
			wantConditions: []operatorv1.OperatorCondition{
				defaultAvailable,
				defaultProgressing,
				{
					Type:               ControllerName + "Controller" + operatorv1.OperatorStatusTypeDegraded,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: transitionTime,
					Message:            "Could not delete mhc",
				},
			},
			wantRequeueAfter: time.Hour,
		},
		{
			name: "Managed Feature Flag is false: mhc deletes but mhc alert fails to delete, an error is returned",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.MachineHealthCheckEnabled: operator.FlagTrue,
						operator.MachineHealthCheckManaged: operator.FlagFalse,
					},
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: defaultConditions,
				},
			},
			clientHook: func(t *testclienthelper.HookingClient) {
				t.WithDeleteHook(func(obj client.Object) error {
					if client.ObjectKeyFromObject(obj).String() == prometheusAlertNamespacedName.String() {
						return errors.New("Could not delete mhc alert")
					}
					return nil
				})
			},
			wantErr: "Could not delete mhc alert",
			wantConditions: []operatorv1.OperatorCondition{
				defaultAvailable,
				defaultProgressing,
				{
					Type:               ControllerName + "Controller" + operatorv1.OperatorStatusTypeDegraded,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: transitionTime,
					Message:            "Could not delete mhc alert",
				},
			},
			wantRequeueAfter: time.Hour,
		},
		{
			name: "Managed Feature Flag is true: dynamic helper ensures resources",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.MachineHealthCheckEnabled: operator.FlagTrue,
						operator.MachineHealthCheckManaged: operator.FlagTrue,
					},
				},
			},
			wantCreates: map[string]int{
				"MachineHealthCheck/openshift-machine-api/aro-machinehealthcheck": 1,
				"PrometheusRule/openshift-machine-api/mhc-remediation-alert":      1,
			},
			wantDeletes:       map[string]int{},
			expectAnnotations: map[string]string{},
			wantConditions:    defaultConditions,
			wantErr:           "",
		},
		{
			name: "Managed Feature Flag is true and cluster is upgrading: sets paused annotation on MHC",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.MachineHealthCheckEnabled: operator.FlagTrue,
						operator.MachineHealthCheckManaged: operator.FlagTrue,
					},
				},
			},
			clusterversion: clusterversionUpgrading,
			wantCreates: map[string]int{
				"MachineHealthCheck/openshift-machine-api/aro-machinehealthcheck": 1,
				"PrometheusRule/openshift-machine-api/mhc-remediation-alert":      1,
			},
			wantDeletes: map[string]int{},
			expectAnnotations: map[string]string{
				MHCPausedAnnotation: "",
			},
			wantErr: "",
		},
		{
			name: "When ensuring resources fails, an error is returned",
			instance: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.MachineHealthCheckEnabled: operator.FlagTrue,
						operator.MachineHealthCheckManaged: operator.FlagTrue,
					},
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: defaultConditions,
				},
			},
			clientHook: func(t *testclienthelper.HookingClient) {
				t.WithCreateHook(func(obj client.Object) error {
					return errors.New("failed to ensure")
				})
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
			clientBuilder := ctrlfake.NewClientBuilder()
			if tt.instance != nil {
				clientBuilder = clientBuilder.WithObjects(tt.instance)
			}
			if tt.clusterversion == nil {
				clientBuilder = clientBuilder.WithObjects(clusterversionDefault)
			} else {
				clientBuilder = clientBuilder.WithObjects(tt.clusterversion)
			}

			var createTally map[string]int
			var deleteTally map[string]int

			client := testclienthelper.NewHookingClient(clientBuilder.Build())

			if tt.clientHook == nil {
				createTally = map[string]int{}
				deleteTally = map[string]int{}
				client = client.WithCreateHook(testclienthelper.TallyCountsAndKey(createTally)).
					WithDeleteHook(testclienthelper.TallyCountsAndKey(deleteTally))
			} else {
				tt.clientHook(client)
			}
			log := logrus.NewEntry(logrus.StandardLogger())
			ch := clienthelper.NewWithClient(log, client)

			ctx := context.Background()

			r := NewReconciler(
				log,
				client,
				ch,
			)

			request := ctrl.Request{}
			request.Name = "cluster"

			result, err := r.Reconcile(ctx, request)

			if tt.wantRequeueAfter != result.RequeueAfter {
				t.Errorf("wanted to requeue after %v but was set to %v", tt.wantRequeueAfter, result.RequeueAfter)
			}

			if tt.instance != nil {
				utilconditions.AssertControllerConditions(t, ctx, r.AROController.Client, tt.wantConditions)
			}

			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			for _, e := range deep.Equal(tt.wantCreates, createTally) {
				t.Error(e)
			}
			for _, e := range deep.Equal(tt.wantDeletes, deleteTally) {
				t.Error(e)
			}

			if tt.expectAnnotations != nil {
				m := &machinev1beta1.MachineHealthCheck{}
				err = ch.GetOne(ctx, mhcNamespacedName, m)
				if err != nil {
					t.Fatal(err)
				}
				annotations := map[string]string{}
				maps.Copy(annotations, m.ObjectMeta.Annotations)
				for _, e := range deep.Equal(tt.expectAnnotations, annotations) {
					t.Error(e)
				}
			}
		})
	}
}
