package machineset

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maofake "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"
	ctrl "sigs.k8s.io/controller-runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestReconciler(t *testing.T) {
	FakeMachineSets := func(replicas0 int32, replicas1 int32, replicas2 int32) []runtime.Object {
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
		return []runtime.Object{workerMachineSet0, workerMachineSet1, workerMachineSet2}
	}

	baseCluster := arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
		Spec: arov1alpha1.ClusterSpec{
			InfraID:  "aro-fake",
			Features: arov1alpha1.FeaturesSpec{ReconcileMachineSet: true},
		},
	}

	tests := []struct {
		name           string
		maodata        []runtime.Object
		mocks          func(maocli *maofake.Clientset)
		wantReplicas   int32
		assertReplicas bool
		wantErr        string
		objectName     string
	}{
		{
			name:           "no worker replicas, machineset-0 modified",
			maodata:        FakeMachineSets(0, 0, 0),
			wantReplicas:   2, // expected replica count after reconcile
			assertReplicas: true,
			wantErr:        "",
			objectName:     "aro-fake-machineset-0",
		},
		{
			name:           "one worker replica, machineset-0 modified",
			maodata:        FakeMachineSets(1, 0, 0),
			wantReplicas:   2,
			assertReplicas: true,
			wantErr:        "",
			objectName:     "aro-fake-machineset-0",
		},
		{
			name:           "two worker replicas, machineset-0 modified",
			maodata:        FakeMachineSets(1, 1, 0),
			wantReplicas:   1,
			assertReplicas: true,
			wantErr:        "",
			objectName:     "aro-fake-machineset-0",
		},
		{
			name:           "three worker replicas, machineset-0 modified",
			maodata:        FakeMachineSets(1, 1, 1),
			wantReplicas:   1,
			assertReplicas: true,
			wantErr:        "",
			objectName:     "aro-fake-machineset-0",
		},
		{
			name:           "three worker replicas in machineset-1, machineset-0 modified",
			maodata:        FakeMachineSets(0, 3, 0),
			wantReplicas:   0,
			assertReplicas: true,
			wantErr:        "",
			objectName:     "aro-fake-machineset-0",
		},
		{
			name:           "two worker replicas in machineset-1, machineset-0 modified",
			maodata:        FakeMachineSets(0, 2, 0),
			wantReplicas:   0,
			assertReplicas: true,
			wantErr:        "",
			objectName:     "aro-fake-machineset-0",
		},
		{
			name:    "machineset-0 not found",
			maodata: FakeMachineSets(2, 0, 0),
			mocks: func(maocli *maofake.Clientset) {
				maocli.PrependReactor("get", "machinesets", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, &kerrors.StatusError{ErrStatus: metav1.Status{
						Message: "machineset-0 not found",
						Reason:  metav1.StatusReasonNotFound,
					}}
				})
			},
			assertReplicas: false,
			wantErr:        "machineset-0 not found",
			objectName:     "aro-fake-machineset-0",
		},
		{
			name:    "machineset-0 not found, object is no longer present",
			maodata: FakeMachineSets(1, 0, 0),
			mocks: func(maocli *maofake.Clientset) {
				maocli.PrependReactor("get", "machinesets", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, &kerrors.StatusError{ErrStatus: metav1.Status{
						Message: "machineset-0 not found, object is no longer present",
						Reason:  metav1.StatusReasonGone,
					}}
				})
			},
			assertReplicas: false,
			wantErr:        "machineset-0 not found, object is no longer present",
			objectName:     "aro-fake-machineset-0",
		},
		{
			name:    "machineset-0 can't be updated",
			maodata: FakeMachineSets(1, 0, 0),
			mocks: func(maocli *maofake.Clientset) {
				maocli.PrependReactor("update", "machinesets", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, &kerrors.StatusError{ErrStatus: metav1.Status{
						Message: "error updating MachineSet, internal server error",
						Reason:  metav1.StatusReasonInternalError,
					}}
				})
			},
			assertReplicas: false,
			wantErr:        "error updating MachineSet, internal server error",
			objectName:     "aro-fake-machineset-0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maocli := maofake.NewSimpleClientset(tt.maodata...)

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

func TestCustomConfiguration(t *testing.T) {
	// Test the reconciler when there is a custom machineset present
	newFakeMao := func(replicas, customReplicas int32) *maofake.Clientset {
		workerMachineSet := &machinev1beta1.MachineSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aro-fake-machineset-0",
				Namespace: machineSetsNamespace,
				Labels: map[string]string{
					"machine.openshift.io/cluster-api-machine-role": "worker",
				},
			},
			Spec: machinev1beta1.MachineSetSpec{
				Replicas: to.Int32Ptr(replicas), // Modify replicas accordingly
			},
		}
		customMachineSet := &machinev1beta1.MachineSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "custom-machineset",
				Namespace: machineSetsNamespace,
				Labels: map[string]string{
					"machine.openshift.io/cluster-api-machine-role": "worker",
				},
			},
			Spec: machinev1beta1.MachineSetSpec{
				Replicas: to.Int32Ptr(customReplicas),
			},
		}
		return maofake.NewSimpleClientset(workerMachineSet, customMachineSet)
	}

	baseCluster := arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
		Spec: arov1alpha1.ClusterSpec{
			InfraID:  "aro-fake",
			Features: arov1alpha1.FeaturesSpec{ReconcileMachineSet: true},
		},
	}

	tests := []struct {
		name         string
		maocli       *maofake.Clientset
		wantReplicas int32
		wantErr      string
		objectName   string
	}{
		// run the same tests, but expect no changes or errors since we have introduced a custom machineset
		{
			name:         "no worker replicas, machineset-0 modified",
			maocli:       newFakeMao(0, 0),
			wantReplicas: 0,
			wantErr:      "",
			objectName:   "aro-fake-machineset-0",
		},
		{
			name:         "one worker replica, machineset-0 modified",
			maocli:       newFakeMao(1, 0),
			wantReplicas: 1,
			wantErr:      "",
			objectName:   "aro-fake-machineset-0",
		},
		{
			name:         "two worker replicas, machineset-0 modified",
			maocli:       newFakeMao(1, 1),
			wantReplicas: 1,
			wantErr:      "",
			objectName:   "aro-fake-machineset-0",
		},
		{
			name:         "three worker replicas, machineset-0 modified",
			maocli:       newFakeMao(3, 0),
			wantReplicas: 3,
			wantErr:      "",
			objectName:   "aro-fake-machineset-0",
		},
		{
			name:         "three worker replicas in machineset-1, machineset-0 modified",
			maocli:       newFakeMao(0, 3),
			wantReplicas: 0,
			wantErr:      "",
			objectName:   "aro-fake-machineset-0",
		},
		{
			name:         "two worker replicas in machineset-1, custom-machineset modified",
			maocli:       newFakeMao(0, 2),
			wantReplicas: 2,
			wantErr:      "",
			objectName:   "custom-machineset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				log:    logrus.NewEntry(logrus.StandardLogger()),
				arocli: arofake.NewSimpleClientset(&baseCluster),
				maocli: tt.maocli,
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

			modifiedMachineset, err := tt.maocli.MachineV1beta1().MachineSets(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			if *modifiedMachineset.Spec.Replicas != tt.wantReplicas {
				t.Error(cmp.Diff(*modifiedMachineset.Spec.Replicas, tt.wantReplicas))
			}
		})
	}
}

func TestFeatureFlag(t *testing.T) {
	newFakeMao := func(replicas int32) *maofake.Clientset {
		workerMachineSet := &machinev1beta1.MachineSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aro-fake-machineset-0",
				Namespace: machineSetsNamespace,
				Labels: map[string]string{
					"machine.openshift.io/cluster-api-machine-role": "worker",
				},
			},
			Spec: machinev1beta1.MachineSetSpec{
				Replicas: to.Int32Ptr(replicas),
			},
		}
		return maofake.NewSimpleClientset(workerMachineSet)
	}

	baseCluster := arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
		Spec: arov1alpha1.ClusterSpec{
			InfraID:  "aro-fake",
			Features: arov1alpha1.FeaturesSpec{ReconcileMachineSet: false},
		},
	}

	tests := []struct {
		name         string
		maocli       *maofake.Clientset
		wantReplicas int32
		objectName   string
		wantErr      string
	}{
		{
			name:         "feature flag is false, replicas are incorrect",
			maocli:       newFakeMao(0),
			wantReplicas: 0,
			wantErr:      "",
			objectName:   "aro-fake-machineset-0",
		},
		{
			name:         "feature flag is false, replicas are correct",
			maocli:       newFakeMao(3),
			wantReplicas: 3,
			wantErr:      "",
			objectName:   "aro-fake-machineset-0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				log:    logrus.NewEntry(logrus.StandardLogger()),
				arocli: arofake.NewSimpleClientset(&baseCluster),
				maocli: tt.maocli,
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

			modifiedMachineset, err := tt.maocli.MachineV1beta1().MachineSets(request.Namespace).Get(ctx, request.Name, metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			if *modifiedMachineset.Spec.Replicas != tt.wantReplicas {
				t.Error(cmp.Diff(*modifiedMachineset.Spec.Replicas, tt.wantReplicas))
			}
		})
	}
}
