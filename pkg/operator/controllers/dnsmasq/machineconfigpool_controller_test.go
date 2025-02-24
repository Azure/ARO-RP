package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
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
		request        ctrl.Request
		wantErrMsg     string
		wantConditions []operatorv1.OperatorCondition
	}{
		{
			name:       "no cluster",
			objects:    []client.Object{},
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
			request:    ctrl.Request{},
			wantErrMsg: "",
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
					ObjectMeta: metav1.ObjectMeta{
						Name:       "custom",
						Finalizers: []string{MachineConfigPoolControllerName},
					},
					Status: mcv1.MachineConfigPoolStatus{},
					Spec:   mcv1.MachineConfigPoolSpec{},
				},
			},
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "",
					Name:      "custom",
				},
			},
			wantErrMsg: `error getting the ClusterVersion: clusterversions.config.openshift.io "version" not found`,
			wantConditions: []operatorv1.OperatorCondition{
				defaultAvailable, defaultProgressing, {
					Type:               MachineConfigPoolControllerName + "ControllerDegraded",
					Status:             "True",
					Message:            `error getting the ClusterVersion: clusterversions.config.openshift.io "version" not found`,
					LastTransitionTime: transitionTime,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createTally := make(map[string]int)
			updateTally := make(map[string]int)

			client := testclienthelper.NewHookingClient(ctrlfake.NewClientBuilder().
				WithObjects(tt.objects...).
				WithStatusSubresource(tt.objects...).
				Build())

			client.WithPostCreateHook(testclienthelper.TallyCountsAndKey(createTally)).WithPostUpdateHook(testclienthelper.TallyCountsAndKey(updateTally))

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

			e, err := testclienthelper.CompareTally(map[string]int{}, createTally)
			if err != nil {
				t.Errorf("create comparison: %v", err)
				for _, i := range e {
					t.Error(i)
				}
			}

			e, err = testclienthelper.CompareTally(map[string]int{}, updateTally)
			if err != nil {
				t.Errorf("update comparison: %v", err)
				for _, i := range e {
					t.Error(i)
				}
			}
		})
	}
}

func TestMachineConfigPoolReconcilerNotUpgrading(t *testing.T) {
	transitionTime := metav1.Time{Time: time.Now()}
	defaultAvailable := utilconditions.ControllerDefaultAvailable(MachineConfigPoolControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(MachineConfigPoolControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(MachineConfigPoolControllerName)
	defaultConditions := []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded}

	tests := []struct {
		name           string
		objects        []client.Object
		request        ctrl.Request
		wantCreated    map[string]int
		wantUpdated    map[string]int
		wantErrMsg     string
		wantConditions []operatorv1.OperatorCondition
	}{
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
			wantCreated: map[string]int{},
			wantUpdated: map[string]int{},
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
			name: "Deleted MachineConfigPool does nothing",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
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
						Name:              "custom",
						Finalizers:        []string{MachineConfigPoolControllerName},
						DeletionTimestamp: &transitionTime,
					},
					Status: mcv1.MachineConfigPoolStatus{},
					Spec:   mcv1.MachineConfigPoolSpec{},
				},
			},
			wantCreated: map[string]int{},
			wantUpdated: map[string]int{},
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
			name: "MachineConfigPool reconciliation create missing DNS MachineConfigs, even when cluster not upgrading",
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
			wantCreated: map[string]int{
				"MachineConfig//99-custom-aro-dns": 1,
			},
			wantUpdated: map[string]int{},
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
			name: "MachineConfigPool reconciliation does not existing DNS MachineConfigs when not updating",
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
				&mcv1.MachineConfig{
					ObjectMeta: metav1.ObjectMeta{Name: "99-custom-aro-dns"},
					Spec:       mcv1.MachineConfigSpec{},
				},
			},
			wantCreated: map[string]int{},
			wantUpdated: map[string]int{},
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
			name: "MachineConfigPool reconciliation updates existing DNS MachineConfigs when force enabled",
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
					ObjectMeta: metav1.ObjectMeta{
						Name:       "custom",
						Finalizers: []string{MachineConfigPoolControllerName},
					},
					Status: mcv1.MachineConfigPoolStatus{},
					Spec:   mcv1.MachineConfigPoolSpec{},
				},
				&mcv1.MachineConfig{
					ObjectMeta: metav1.ObjectMeta{Name: "99-custom-aro-dns"},
					Spec:       mcv1.MachineConfigSpec{},
				},
			},
			wantCreated: map[string]int{},
			wantUpdated: map[string]int{
				"//99-custom-aro-dns": 1,
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
			createTally := make(map[string]int)
			updateTally := make(map[string]int)

			client := testclienthelper.NewHookingClient(ctrlfake.NewClientBuilder().
				WithObjects(tt.objects...).
				WithObjects(&configv1.ClusterVersion{
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
				}).
				WithStatusSubresource(tt.objects...).
				Build())

			client.WithPostCreateHook(testclienthelper.TallyCountsAndKey(createTally)).WithPostUpdateHook(testclienthelper.TallyCountsAndKey(updateTally))

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

			e, err := testclienthelper.CompareTally(tt.wantCreated, createTally)
			if err != nil {
				t.Errorf("create comparison: %v", err)
				for _, i := range e {
					t.Error(i)
				}
			}

			e, err = testclienthelper.CompareTally(tt.wantUpdated, updateTally)
			if err != nil {
				t.Errorf("update comparison: %v", err)
				for _, i := range e {
					t.Error(i)
				}
			}
		})
	}
}

func TestMachineConfigPoolReconcilerClusterUpgrading(t *testing.T) {
	defaultAvailable := utilconditions.ControllerDefaultAvailable(MachineConfigPoolControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(MachineConfigPoolControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(MachineConfigPoolControllerName)
	defaultConditions := []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded}

	tests := []struct {
		name           string
		objects        []client.Object
		request        ctrl.Request
		wantCreated    map[string]int
		wantUpdated    map[string]int
		wantErrMsg     string
		wantConditions []operatorv1.OperatorCondition
	}{
		{
			name: "MachineConfigPool reconciliation create missing DNS MachineConfigs",
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
			wantCreated: map[string]int{
				"MachineConfig//99-custom-aro-dns": 1,
			},
			wantUpdated: map[string]int{},
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
			name: "MachineConfigPool reconciliation updates existing DNS MachineConfigs",
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
				&mcv1.MachineConfig{
					ObjectMeta: metav1.ObjectMeta{Name: "99-custom-aro-dns"},
					Spec:       mcv1.MachineConfigSpec{},
				},
			},
			wantCreated: map[string]int{},
			wantUpdated: map[string]int{
				"//99-custom-aro-dns": 1,
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
			createTally := make(map[string]int)
			updateTally := make(map[string]int)

			client := testclienthelper.NewHookingClient(ctrlfake.NewClientBuilder().
				WithObjects(tt.objects...).
				WithObjects(&configv1.ClusterVersion{
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
				}).
				WithStatusSubresource(tt.objects...).
				Build())

			client.WithPostCreateHook(testclienthelper.TallyCountsAndKey(createTally)).WithPostUpdateHook(testclienthelper.TallyCountsAndKey(updateTally))

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

			e, err := testclienthelper.CompareTally(tt.wantCreated, createTally)
			if err != nil {
				t.Errorf("create comparison: %v", err)
				for _, i := range e {
					t.Error(i)
				}
			}

			e, err = testclienthelper.CompareTally(tt.wantUpdated, updateTally)
			if err != nil {
				t.Errorf("update comparison: %v", err)
				for _, i := range e {
					t.Error(i)
				}
			}
		})
	}
}
