package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/go-test/deep"
	operatorv1 "github.com/openshift/api/operator/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	testdh "github.com/Azure/ARO-RP/test/util/dynamichelper"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestClusterReconciler(t *testing.T) {
	transitionTime := metav1.Time{Time: time.Now()}
	defaultAvailable := utilconditions.ControllerDefaultAvailable(ClusterControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(ClusterControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(ClusterControllerName)
	defaultConditions := []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded}

	tests := []struct {
		name           string
		objects        []client.Object
		request        ctrl.Request
		wantErrMsg     string
		wantConditions []operatorv1.OperatorCondition
		wantCreated    map[string]int
		wantUpdated    map[string]int
		wantDeleted    map[string]int
	}{
		{
			name:        "no cluster",
			objects:     []client.Object{},
			request:     ctrl.Request{},
			wantErrMsg:  "clusters.aro.openshift.io \"cluster\" not found",
			wantCreated: map[string]int{},
			wantUpdated: map[string]int{},
			wantDeleted: map[string]int{},
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
							controllerEnabled: "false",
						},
					},
				},
			},
			request:        ctrl.Request{},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
			wantCreated:    map[string]int{},
			wantUpdated:    map[string]int{},
			wantDeleted:    map[string]int{},
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
							controllerEnabled: "true",
						},
					},
				},
			},
			request:        ctrl.Request{},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
			wantCreated:    map[string]int{},
			wantUpdated:    map[string]int{},
			wantDeleted:    map[string]int{},
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
							controllerEnabled: "true",
						},
					},
				},
				&mcv1.MachineConfigPool{
					ObjectMeta: metav1.ObjectMeta{Name: "master"},
					Status:     mcv1.MachineConfigPoolStatus{},
					Spec:       mcv1.MachineConfigPoolSpec{},
				},
			},
			request:        ctrl.Request{},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
			wantCreated: map[string]int{
				"MachineConfig//99-master-aro-dns": 1,
			},
			wantUpdated: map[string]int{},
			wantDeleted: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, log := testlog.New()

			clientFake := ctrlfake.NewClientBuilder().
				WithObjects(tt.objects...).
				Build()

			deployedObjects := map[string]int{}
			deletedObjects := map[string]int{}
			updatedObjects := map[string]int{}
			wrappedClient := testdh.NewRedirectingClient(clientFake).
				WithCreateHook(testdh.TallyCountsAndKey(deployedObjects)).
				WithDeleteHook(testdh.TallyCounts(deletedObjects)).
				WithUpdateHook(testdh.TallyCounts(updatedObjects))
			dh := dynamichelper.NewWithClient(log, wrappedClient)

			r := NewClusterReconciler(
				logrus.NewEntry(logrus.StandardLogger()),
				clientFake,
				dh,
			)
			ctx := context.Background()
			_, err := r.Reconcile(ctx, tt.request)

			for _, v := range deep.Equal(deployedObjects, tt.wantCreated) {
				t.Errorf("created does not match: %s", v)
			}
			for _, v := range deep.Equal(deletedObjects, tt.wantDeleted) {
				t.Errorf("deleted does not match: %s", v)
			}
			for _, v := range deep.Equal(updatedObjects, tt.wantUpdated) {
				t.Errorf("updated does not match: %s", v)
			}
			utilerror.AssertErrorMessage(t, err, tt.wantErrMsg)
			utilconditions.AssertControllerConditions(t, ctx, clientFake, tt.wantConditions)
		})
	}
}
