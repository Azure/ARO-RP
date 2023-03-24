package machineset

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func TestReconciler(t *testing.T) {
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
		name           string
		objectName     string
		machinesets    []client.Object
		wantReplicas   int32
		featureFlag    bool
		assertReplicas bool
		wantErr        string
	}{
		{
			name:           "no worker replicas, machineset-0 modified",
			objectName:     "aro-fake-machineset-0",
			machinesets:    fakeMachineSets(0, 0, 0),
			wantReplicas:   2,
			featureFlag:    true,
			assertReplicas: true,
			wantErr:        "",
		},
		{
			name:           "no worker replicas, feature flag is false",
			objectName:     "aro-fake-machineset-0",
			machinesets:    fakeMachineSets(0, 0, 0),
			wantReplicas:   0,
			featureFlag:    false,
			assertReplicas: true,
			wantErr:        "",
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
			wantReplicas:   0,
			featureFlag:    true,
			assertReplicas: true,
			wantErr:        "",
		},
		{
			name:           "one worker replica, machineset-0 modified",
			objectName:     "aro-fake-machineset-0",
			machinesets:    fakeMachineSets(1, 0, 0),
			wantReplicas:   2,
			featureFlag:    true,
			assertReplicas: true,
			wantErr:        "",
		},
		{
			name:           "two worker replicas, machineset-0 modified",
			objectName:     "aro-fake-machineset-0",
			machinesets:    fakeMachineSets(1, 1, 0),
			wantReplicas:   1,
			featureFlag:    true,
			assertReplicas: true,
			wantErr:        "",
		},
		{
			name:           "two worker replicas in machineset-1, machineset-0 modified",
			objectName:     "aro-fake-machineset-0",
			machinesets:    fakeMachineSets(0, 2, 0),
			wantReplicas:   0,
			featureFlag:    true,
			assertReplicas: true,
			wantErr:        "",
		},
		{
			name:           "machineset-0 not found",
			objectName:     "aro-fake-machineset-0",
			featureFlag:    true,
			assertReplicas: false,
			wantErr:        `machinesets.machine.openshift.io "aro-fake-machineset-0" not found`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: arov1alpha1.ClusterSpec{
					InfraID: "aro-fake",
					OperatorFlags: arov1alpha1.OperatorFlags{
						ControllerEnabled: strconv.FormatBool(tt.featureFlag),
					},
				},
			}

			clientFake := ctrlfake.NewClientBuilder().WithObjects(instance).WithObjects(tt.machinesets...).Build()

			r := &Reconciler{
				log:    logrus.NewEntry(logrus.StandardLogger()),
				client: clientFake,
			}

			request := ctrl.Request{}
			request.Name = tt.objectName
			request.Namespace = machineSetsNamespace
			ctx := context.Background()

			_, err := r.Reconcile(ctx, request)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if tt.assertReplicas {
				modifiedMachineset := &machinev1beta1.MachineSet{}
				err = r.client.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: machineSetsNamespace}, modifiedMachineset)
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
