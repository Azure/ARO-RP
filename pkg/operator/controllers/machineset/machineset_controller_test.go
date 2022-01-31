package machineset

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maofake "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"
	ctrl "sigs.k8s.io/controller-runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestReconciler(t *testing.T) {
	fakeMachineSets := func(replicas0 int32, replicas1 int32, replicas2 int32) []kruntime.Object {
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
		return []kruntime.Object{workerMachineSet0, workerMachineSet1, workerMachineSet2}
	}

	tests := []struct {
		name           string
		objectName     string
		machinesets    []kruntime.Object
		mocks          func(maocli *maofake.Clientset)
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
			machinesets: func() []kruntime.Object {
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
			name:        "machineset-0 not found",
			objectName:  "aro-fake-machineset-0",
			machinesets: fakeMachineSets(2, 0, 0),
			mocks: func(maocli *maofake.Clientset) {
				maocli.PrependReactor("get", "machinesets", func(action ktesting.Action) (handled bool, ret kruntime.Object, err error) {
					return true, nil, &kerrors.StatusError{ErrStatus: metav1.Status{
						Message: "machineset-0 not found",
						Reason:  metav1.StatusReasonNotFound,
					}}
				})
			},
			featureFlag:    true,
			assertReplicas: false,
			wantErr:        "machineset-0 not found",
		},
		{
			name:        "get machinesets failed with error",
			objectName:  "aro-fake-machineset-0",
			machinesets: fakeMachineSets(1, 0, 0),
			mocks: func(maocli *maofake.Clientset) {
				maocli.PrependReactor("get", "machinesets", func(action ktesting.Action) (handled bool, ret kruntime.Object, err error) {
					return true, nil, errors.New("fake error")
				})
			},
			featureFlag:    true,
			assertReplicas: false,
			wantErr:        "fake error",
		},
		{
			name:        "machineset-0 can't be updated",
			objectName:  "aro-fake-machineset-0",
			machinesets: fakeMachineSets(1, 0, 0),
			mocks: func(maocli *maofake.Clientset) {
				maocli.PrependReactor("update", "machinesets", func(action ktesting.Action) (handled bool, ret kruntime.Object, err error) {
					return true, nil, errors.New("fake error from update")
				})
			},
			featureFlag:    true,
			assertReplicas: false,
			wantErr:        "fake error from update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseCluster := arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: arov1alpha1.ClusterSpec{
					InfraID: "aro-fake",
					OperatorFlags: arov1alpha1.OperatorFlags{
						ControllerEnabled: strconv.FormatBool(tt.featureFlag),
					},
				},
			}

			maocli := maofake.NewSimpleClientset(tt.machinesets...)

			if tt.mocks != nil {
				tt.mocks(maocli)
			}

			r := &Reconciler{
				log:    logrus.NewEntry(logrus.StandardLogger()),
				arocli: arofake.NewSimpleClientset(&baseCluster),
				maocli: maocli,
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
				modifiedMachineset, err := maocli.MachineV1beta1().MachineSets(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
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
