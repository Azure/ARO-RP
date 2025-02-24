package machineset

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestReconciler(t *testing.T) {
	transitionTime := metav1.Time{Time: time.Now()}
	defaultAvailable := utilconditions.ControllerDefaultAvailable(ControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(ControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(ControllerName)
	defaultConditions := []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded}

	fakeMachineSets := func(replicas0 int32, replicas1 int32, replicas2 int32) []client.Object {
		workerMachineSet0 := &machinev1beta1.MachineSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aro-fake-machineset-0",
				Namespace: machineSetsNamespace,
				Labels: map[string]string{
					"machine.openshift.io/cluster-api-machine-role": "worker",
				},
			},
			Spec: machinev1beta1.MachineSetSpec{
				Replicas: to.Int32Ptr(replicas0), // Modify replicas accordingly
			},
		}
		workerMachineSet1 := &machinev1beta1.MachineSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aro-fake-machineset-1",
				Namespace: machineSetsNamespace,
				Labels: map[string]string{
					"machine.openshift.io/cluster-api-machine-role": "worker",
				},
			},
			Spec: machinev1beta1.MachineSetSpec{
				Replicas: to.Int32Ptr(replicas1),
			},
		}
		workerMachineSet2 := &machinev1beta1.MachineSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aro-fake-machineset-2",
				Namespace: machineSetsNamespace,
				Labels: map[string]string{
					"machine.openshift.io/cluster-api-machine-role": "worker",
				},
			},
			Spec: machinev1beta1.MachineSetSpec{
				Replicas: to.Int32Ptr(replicas2),
			},
		}
		return []client.Object{workerMachineSet0, workerMachineSet1, workerMachineSet2}
	}

	tests := []struct {
		name            string
		objectName      string
		machinesets     []client.Object
		wantReplicas    int32
		featureFlag     bool
		assertReplicas  bool
		wantErr         string
		startConditions []operatorv1.OperatorCondition
		wantConditions  []operatorv1.OperatorCondition
	}{
		{
			name:            "no worker replicas, machineset-0 modified",
			objectName:      "aro-fake-machineset-0",
			machinesets:     fakeMachineSets(0, 0, 0),
			wantReplicas:    2,
			featureFlag:     true,
			assertReplicas:  true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
		},
		{
			name:            "no worker replicas, feature flag is false",
			objectName:      "aro-fake-machineset-0",
			machinesets:     fakeMachineSets(0, 0, 0),
			wantReplicas:    0,
			featureFlag:     false,
			assertReplicas:  true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
		},
		{
			name:       "no worker replicas, custom machineset is present",
			objectName: "aro-fake-machineset-0",
			machinesets: func() []client.Object {
				return append(
					fakeMachineSets(0, 0, 0),
					&machinev1beta1.MachineSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "custom-machineset",
							Namespace: machineSetsNamespace,
							Labels: map[string]string{
								"machine.openshift.io/cluster-api-machine-role": "worker",
							},
						},
						Spec: machinev1beta1.MachineSetSpec{
							Replicas: to.Int32Ptr(0),
						},
					},
				)
			}(),
			wantReplicas:    0,
			featureFlag:     true,
			assertReplicas:  true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
		},
		{
			name:           "one worker replica, machineset-0 modified",
			objectName:     "aro-fake-machineset-0",
			machinesets:    fakeMachineSets(1, 0, 0),
			wantReplicas:   2,
			featureFlag:    true,
			assertReplicas: true,
			wantErr:        "",
			startConditions: []operatorv1.OperatorCondition{
				defaultAvailable,
				defaultProgressing,
				{
					Type:               ControllerName + "Controller" + operatorv1.OperatorStatusTypeDegraded,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: transitionTime,
					Message:            `machinesets.machine.openshift.io "aro-fake-machineset-0" not found`,
				},
			},
			wantConditions: defaultConditions,
		},
		{
			name:            "two worker replicas, machineset-0 modified",
			objectName:      "aro-fake-machineset-0",
			machinesets:     fakeMachineSets(1, 1, 0),
			wantReplicas:    1,
			featureFlag:     true,
			assertReplicas:  true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
		},
		{
			name:            "two worker replicas in machineset-1, machineset-0 modified",
			objectName:      "aro-fake-machineset-0",
			machinesets:     fakeMachineSets(0, 2, 0),
			wantReplicas:    0,
			featureFlag:     true,
			assertReplicas:  true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
		},
		{
			name:            "machineset-0 not found",
			objectName:      "aro-fake-machineset-0",
			featureFlag:     true,
			assertReplicas:  false,
			wantErr:         `machinesets.machine.openshift.io "aro-fake-machineset-0" not found`,
			startConditions: defaultConditions,
			wantConditions: []operatorv1.OperatorCondition{
				defaultAvailable,
				defaultProgressing,
				{
					Type:               ControllerName + "Controller" + operatorv1.OperatorStatusTypeDegraded,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: transitionTime,
					Message:            `machinesets.machine.openshift.io "aro-fake-machineset-0" not found`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: arov1alpha1.ClusterSpec{
					InfraID: "aro-fake",
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.MachineSetEnabled: strconv.FormatBool(tt.featureFlag),
					},
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: tt.startConditions,
				},
			}

			clientFake := ctrlfake.NewClientBuilder().
				WithObjects(instance).
				WithStatusSubresource(instance).
				WithObjects(tt.machinesets...).
				Build()

			r := NewReconciler(logrus.NewEntry(logrus.StandardLogger()), clientFake)

			request := ctrl.Request{}
			request.Name = tt.objectName
			request.Namespace = machineSetsNamespace
			ctx := context.Background()

			_, err := r.Reconcile(ctx, request)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			utilconditions.AssertControllerConditions(t, ctx, clientFake, tt.wantConditions)

			if tt.assertReplicas {
				modifiedMachineset := &machinev1beta1.MachineSet{}
				err = r.Client.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: machineSetsNamespace}, modifiedMachineset)
				if err != nil {
					t.Error(err)
				}

				if *modifiedMachineset.Spec.Replicas != tt.wantReplicas {
					t.Error(cmp.Diff(*modifiedMachineset.Spec.Replicas, tt.wantReplicas))
				}
			}
		})
	}
}
