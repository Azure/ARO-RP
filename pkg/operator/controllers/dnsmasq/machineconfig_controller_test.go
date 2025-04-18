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

func TestMachineConfigReconciler(t *testing.T) {
	transitionTime := metav1.Time{Time: time.Now()}
	defaultAvailable := utilconditions.ControllerDefaultAvailable(MachineConfigControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(MachineConfigControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(MachineConfigControllerName)
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
			request:    ctrl.Request{},
			wantErrMsg: `error getting the ClusterVersion: clusterversions.config.openshift.io "version" not found`,
			wantConditions: []operatorv1.OperatorCondition{
				defaultAvailable, defaultProgressing, {
					Type:               MachineConfigControllerName + "ControllerDegraded",
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

			r := NewMachineConfigReconciler(
				logrus.NewEntry(logrus.StandardLogger()),
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

func TestMachineConfigReconcilerNotUpgrading(t *testing.T) {
	defaultAvailable := utilconditions.ControllerDefaultAvailable(MachineConfigControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(MachineConfigControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(MachineConfigControllerName)
	defaultConditions := []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded}

	tests := []struct {
		name           string
		objects        []client.Object
		wantCreated    map[string]int
		wantUpdated    map[string]int
		request        ctrl.Request
		wantErrMsg     string
		wantConditions []operatorv1.OperatorCondition
	}{
		{
			name: "valid MachineConfigPool for MachineConfig creates MachineConfig, even if cluster not updating",
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
					ObjectMeta: metav1.ObjectMeta{Name: "custom"},
					Status:     mcv1.MachineConfigPoolStatus{},
					Spec:       mcv1.MachineConfigPoolSpec{},
				},
			},
			wantCreated: map[string]int{
				"MachineConfig//99-custom-aro-dns": 1,
			},
			wantUpdated: map[string]int{},
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "",
					Name:      "99-custom-aro-dns",
				},
			},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "valid MachineConfigPool for MachineConfig does not update existing MachineConfig while cluster not upgrading",
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
					ObjectMeta: metav1.ObjectMeta{Name: "custom"},
					Status:     mcv1.MachineConfigPoolStatus{},
					Spec:       mcv1.MachineConfigPoolSpec{},
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
					Name:      "99-custom-aro-dns",
				},
			},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "valid MachineConfigPool for MachineConfig updates existing MachineConfig  when reconciliation forced",
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
					ObjectMeta: metav1.ObjectMeta{Name: "custom"},
					Status:     mcv1.MachineConfigPoolStatus{},
					Spec:       mcv1.MachineConfigPoolSpec{},
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
					Name:      "99-custom-aro-dns",
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

			cluster := &configv1.ClusterVersion{
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
			}
			client := testclienthelper.NewHookingClient(ctrlfake.NewClientBuilder().
				WithObjects(tt.objects...).
				WithStatusSubresource(tt.objects...).
				WithObjects(cluster).
				Build())

			client.WithPostCreateHook(testclienthelper.TallyCountsAndKey(createTally)).WithPostUpdateHook(testclienthelper.TallyCountsAndKey(updateTally))

			log := logrus.NewEntry(logrus.StandardLogger())
			ch := clienthelper.NewWithClient(log, client)

			r := NewMachineConfigReconciler(
				logrus.NewEntry(logrus.StandardLogger()),
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

func TestMachineConfigReconcilerClusterUpgrading(t *testing.T) {
	transitionTime := metav1.Time{Time: time.Now()}
	defaultAvailable := utilconditions.ControllerDefaultAvailable(MachineConfigControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(MachineConfigControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(MachineConfigControllerName)
	defaultConditions := []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded}

	tests := []struct {
		name           string
		objects        []client.Object
		wantCreated    map[string]int
		wantUpdated    map[string]int
		request        ctrl.Request
		wantErrMsg     string
		wantConditions []operatorv1.OperatorCondition
	}{
		{
			name: "no MachineConfigPool for MachineConfig does nothing (cluster upgrading)",
			objects: []client.Object{
				&arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
					Status: arov1alpha1.ClusterStatus{
						Conditions: []operatorv1.OperatorCondition{
							defaultAvailable,
							defaultProgressing,
							{
								Type:               MachineConfigControllerName + "Controller" + operatorv1.OperatorStatusTypeDegraded,
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
					Name:      "99-custom-aro-dns",
				},
			},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "valid MachineConfigPool for MachineConfig creates MachineConfig while cluster upgrading",
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
					ObjectMeta: metav1.ObjectMeta{Name: "custom"},
					Status:     mcv1.MachineConfigPoolStatus{},
					Spec:       mcv1.MachineConfigPoolSpec{},
				},
			},
			wantCreated: map[string]int{
				"MachineConfig//99-custom-aro-dns": 1,
			},
			wantUpdated: map[string]int{},
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "",
					Name:      "99-custom-aro-dns",
				},
			},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "valid MachineConfigPool for MachineConfig updates existing MachineConfig while cluster upgrading",
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
					ObjectMeta: metav1.ObjectMeta{Name: "custom"},
					Status:     mcv1.MachineConfigPoolStatus{},
					Spec:       mcv1.MachineConfigPoolSpec{},
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
					Name:      "99-custom-aro-dns",
				},
			},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
		{
			name: "changes for a deleted MachineConfigPool do nothing",
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
						Name:              "custom",
						DeletionTimestamp: &transitionTime,
						Finalizers:        []string{"test-finalizer"},
					},
					Status: mcv1.MachineConfigPoolStatus{},
					Spec:   mcv1.MachineConfigPoolSpec{},
				},
			},
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "",
					Name:      "99-custom-aro-dns",
				},
			},
			wantCreated:    map[string]int{},
			wantUpdated:    map[string]int{},
			wantErrMsg:     "",
			wantConditions: defaultConditions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createTally := make(map[string]int)
			updateTally := make(map[string]int)

			clusterversion := &configv1.ClusterVersion{
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
			}

			client := testclienthelper.NewHookingClient(ctrlfake.NewClientBuilder().
				WithObjects(tt.objects...).
				WithStatusSubresource(tt.objects...).
				WithObjects(clusterversion).
				Build())

			client.WithPostCreateHook(testclienthelper.TallyCountsAndKey(createTally)).WithPostUpdateHook(testclienthelper.TallyCountsAndKey(updateTally))

			log := logrus.NewEntry(logrus.StandardLogger())
			ch := clienthelper.NewWithClient(log, client)

			r := NewMachineConfigReconciler(
				logrus.NewEntry(logrus.StandardLogger()),
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
