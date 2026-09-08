package machinehealthcheck

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"

	configv1 "github.com/openshift/api/config/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

// Test reconcile function
func TestMachineHealthCheckReconciler(t *testing.T) {
	defaultAvailable := utilconditions.ControllerDefaultAvailable(ControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(ControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(ControllerName)

	defaultConditions := []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded}

	clusterversionDefault := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{
				{
					State:   configv1.CompletedUpdate,
					Version: "4.18.30",
				},
			},
			Conditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:   configv1.OperatorProgressing,
					Status: configv1.ConditionFalse,
				},
			},
		},
	}
	clusterversionUpgrading := clusterversionDefault.DeepCopy()
	clusterversionUpgrading.Status.History = []configv1.UpdateHistory{
		{
			State:   configv1.PartialUpdate,
			Version: "4.19.0",
		},
		{
			State:   configv1.CompletedUpdate,
			Version: "4.18.30",
		},
	}

	// ARO-26990: MCO rollout — Progressing=True but no new version in history.
	// IsClusterUpgrading must return false for this case.
	clusterversionMCORollout := clusterversionDefault.DeepCopy()
	clusterversionMCORollout.Status.Conditions = []configv1.ClusterOperatorStatusCondition{
		{
			Type:   configv1.OperatorProgressing,
			Status: configv1.ConditionTrue,
		},
	}

	mhcKey := types.NamespacedName{Namespace: "openshift-machine-api", Name: "aro-machinehealthcheck"}

	type test struct {
		name             string
		instance         *arov1alpha1.Cluster
		clusterversion   *configv1.ClusterVersion
		existingMHC      *machinev1beta1.MachineHealthCheck
		wantConditions   []operatorv1.OperatorCondition
		wantErr          string
		wantRequeueAfter time.Duration
		assertMHC        func(t *testing.T, ctx context.Context, r *Reconciler)
	}

	for _, tt := range []*test{
		{
			name:           "Failure to get instance",
			wantConditions: defaultConditions,
			wantErr:        `clusters.aro.openshift.io "cluster" not found`,
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
			wantConditions: defaultConditions,
			wantErr:        "",
		},
		{
			name: "Managed Feature Flag is false: ensure MHC is deleted",
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
			existingMHC: &machinev1beta1.MachineHealthCheck{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-machinehealthcheck",
					Namespace: "openshift-machine-api",
				},
				Spec: machinev1beta1.MachineHealthCheckSpec{
					MaxUnhealthy: pointerutils.ToPtr(intstr.FromInt32(1)),
				},
			},
			wantConditions: defaultConditions,
			wantErr:        "",
			assertMHC: func(t *testing.T, ctx context.Context, r *Reconciler) {
				t.Helper()
				mhc := &machinev1beta1.MachineHealthCheck{}
				err := r.Client.Get(ctx, mhcKey, mhc)
				if err == nil {
					t.Fatal("expected MHC to be deleted, but it still exists")
				}
			},
		},
		{
			name: "Managed Feature Flag is true: MHC is created with defaults",
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
			wantConditions: defaultConditions,
			wantErr:        "",
			assertMHC: func(t *testing.T, ctx context.Context, r *Reconciler) {
				t.Helper()
				mhc := &machinev1beta1.MachineHealthCheck{}
				if err := r.Client.Get(ctx, mhcKey, mhc); err != nil {
					t.Fatalf("expected MHC to be created: %v", err)
				}
				if mhc.Spec.MaxUnhealthy == nil {
					t.Fatal("expected maxUnhealthy to be set")
				}
				if mhc.Spec.MaxUnhealthy.IntValue() != 1 {
					t.Errorf("expected maxUnhealthy=1, got %v", mhc.Spec.MaxUnhealthy)
				}
			},
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
			wantErr:        "",
			assertMHC: func(t *testing.T, ctx context.Context, r *Reconciler) {
				t.Helper()
				mhc := &machinev1beta1.MachineHealthCheck{}
				if err := r.Client.Get(ctx, mhcKey, mhc); err != nil {
					t.Fatalf("expected MHC to exist: %v", err)
				}
				if _, ok := mhc.Annotations[MHCPausedAnnotation]; !ok {
					t.Error("expected paused annotation to be set during upgrade")
				}
			},
		},
		{
			// ARO-26990: an ARO MCO config rollout sets Progressing=True but does not
			// push a new version into history. MHC must NOT be paused in this case.
			name: "ARO-26990: MCO rollout (Progressing=True, history[0]=Completed): does not pause MHC",
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
			clusterversion: clusterversionMCORollout,
			wantErr:        "",
			assertMHC: func(t *testing.T, ctx context.Context, r *Reconciler) {
				t.Helper()
				mhc := &machinev1beta1.MachineHealthCheck{}
				if err := r.Client.Get(ctx, mhcKey, mhc); err != nil {
					t.Fatalf("expected MHC to exist: %v", err)
				}
				if _, ok := mhc.Annotations[MHCPausedAnnotation]; ok {
					t.Error("expected paused annotation NOT to be set during MCO rollout")
				}
			},
		},
		{
			name: "Existing MHC with customer maxUnhealthy override is preserved",
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
			existingMHC: &machinev1beta1.MachineHealthCheck{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-machinehealthcheck",
					Namespace: "openshift-machine-api",
				},
				Spec: machinev1beta1.MachineHealthCheckSpec{
					MaxUnhealthy: pointerutils.ToPtr(intstr.FromInt32(2)),
				},
			},
			wantConditions: defaultConditions,
			wantErr:        "",
			assertMHC: func(t *testing.T, ctx context.Context, r *Reconciler) {
				t.Helper()
				mhc := &machinev1beta1.MachineHealthCheck{}
				if err := r.Client.Get(ctx, mhcKey, mhc); err != nil {
					t.Fatalf("expected MHC to exist: %v", err)
				}
				if mhc.Spec.MaxUnhealthy == nil {
					t.Fatal("expected maxUnhealthy to be set")
				}
				if mhc.Spec.MaxUnhealthy.IntValue() != 2 {
					t.Errorf("expected customer maxUnhealthy=2 to be preserved, got %v", mhc.Spec.MaxUnhealthy)
				}
			},
		},
		{
			name: "Existing MHC with integer 0 maxUnhealthy is defaulted to 1",
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
			existingMHC: &machinev1beta1.MachineHealthCheck{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-machinehealthcheck",
					Namespace: "openshift-machine-api",
				},
				Spec: machinev1beta1.MachineHealthCheckSpec{
					MaxUnhealthy: pointerutils.ToPtr(intstr.FromInt32(0)),
				},
			},
			wantConditions: defaultConditions,
			wantErr:        "",
			assertMHC: func(t *testing.T, ctx context.Context, r *Reconciler) {
				t.Helper()
				mhc := &machinev1beta1.MachineHealthCheck{}
				if err := r.Client.Get(ctx, mhcKey, mhc); err != nil {
					t.Fatalf("expected MHC to exist: %v", err)
				}
				if mhc.Spec.MaxUnhealthy == nil {
					t.Fatal("expected maxUnhealthy to be defaulted")
				}
				if mhc.Spec.MaxUnhealthy.IntValue() != 1 {
					t.Errorf("expected maxUnhealthy=1 default, got %v", mhc.Spec.MaxUnhealthy)
				}
			},
		},
		{
			name: "Existing MHC with string 0 maxUnhealthy is defaulted to 1",
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
			existingMHC: &machinev1beta1.MachineHealthCheck{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-machinehealthcheck",
					Namespace: "openshift-machine-api",
				},
				Spec: machinev1beta1.MachineHealthCheckSpec{
					MaxUnhealthy: pointerutils.ToPtr(intstr.FromString("0")),
				},
			},
			wantConditions: defaultConditions,
			wantErr:        "",
			assertMHC: func(t *testing.T, ctx context.Context, r *Reconciler) {
				t.Helper()
				mhc := &machinev1beta1.MachineHealthCheck{}
				if err := r.Client.Get(ctx, mhcKey, mhc); err != nil {
					t.Fatalf("expected MHC to exist: %v", err)
				}
				if mhc.Spec.MaxUnhealthy == nil {
					t.Fatal("expected maxUnhealthy to be defaulted")
				}
				if mhc.Spec.MaxUnhealthy.IntValue() != 1 {
					t.Errorf("expected maxUnhealthy=1 default, got %v", mhc.Spec.MaxUnhealthy)
				}
			},
		},
		{
			name: "Existing MHC with 100% maxUnhealthy is preserved",
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
			existingMHC: &machinev1beta1.MachineHealthCheck{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-machinehealthcheck",
					Namespace: "openshift-machine-api",
				},
				Spec: machinev1beta1.MachineHealthCheckSpec{
					MaxUnhealthy: pointerutils.ToPtr(intstr.FromString("100%")),
				},
			},
			wantConditions: defaultConditions,
			wantErr:        "",
			assertMHC: func(t *testing.T, ctx context.Context, r *Reconciler) {
				t.Helper()
				mhc := &machinev1beta1.MachineHealthCheck{}
				if err := r.Client.Get(ctx, mhcKey, mhc); err != nil {
					t.Fatalf("expected MHC to exist: %v", err)
				}
				if mhc.Spec.MaxUnhealthy == nil {
					t.Fatal("expected maxUnhealthy to still be set")
				}
				if mhc.Spec.MaxUnhealthy.StrVal != "100%" {
					t.Errorf("expected maxUnhealthy to remain 100%%, got %v", mhc.Spec.MaxUnhealthy)
				}
			},
		},
		{
			name: "Existing MHC with missing required selectors has them restored",
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
			existingMHC: &machinev1beta1.MachineHealthCheck{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-machinehealthcheck",
					Namespace: "openshift-machine-api",
				},
				Spec: machinev1beta1.MachineHealthCheckSpec{
					MaxUnhealthy: pointerutils.ToPtr(intstr.FromInt32(1)),
					Selector:     metav1.LabelSelector{},
				},
			},
			wantConditions: defaultConditions,
			wantErr:        "",
			assertMHC: func(t *testing.T, ctx context.Context, r *Reconciler) {
				t.Helper()
				mhc := &machinev1beta1.MachineHealthCheck{}
				if err := r.Client.Get(ctx, mhcKey, mhc); err != nil {
					t.Fatalf("expected MHC to exist: %v", err)
				}
				if len(mhc.Spec.Selector.MatchExpressions) != 2 {
					t.Fatalf("expected 2 required matchExpressions, got %d", len(mhc.Spec.Selector.MatchExpressions))
				}
			},
		},
		{
			name: "Existing MHC with required selectors plus custom ones preserves all",
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
			existingMHC: &machinev1beta1.MachineHealthCheck{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-machinehealthcheck",
					Namespace: "openshift-machine-api",
				},
				Spec: machinev1beta1.MachineHealthCheckSpec{
					MaxUnhealthy: pointerutils.ToPtr(intstr.FromInt32(1)),
					Selector: metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "machine.openshift.io/cluster-api-machine-role",
								Operator: metav1.LabelSelectorOpNotIn,
								Values:   []string{"master"},
							},
							{
								Key:      "machine.openshift.io/cluster-api-machineset",
								Operator: metav1.LabelSelectorOpExists,
							},
							{
								Key:      "custom-label",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{"custom-value"},
							},
						},
					},
				},
			},
			wantConditions: defaultConditions,
			wantErr:        "",
			assertMHC: func(t *testing.T, ctx context.Context, r *Reconciler) {
				t.Helper()
				mhc := &machinev1beta1.MachineHealthCheck{}
				if err := r.Client.Get(ctx, mhcKey, mhc); err != nil {
					t.Fatalf("expected MHC to exist: %v", err)
				}
				if len(mhc.Spec.Selector.MatchExpressions) != 3 {
					t.Fatalf("expected 3 matchExpressions (2 required + 1 custom), got %d", len(mhc.Spec.Selector.MatchExpressions))
				}
			},
		},
		{
			name: "Existing MHC with corrupted selector values has them corrected",
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
			existingMHC: &machinev1beta1.MachineHealthCheck{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-machinehealthcheck",
					Namespace: "openshift-machine-api",
				},
				Spec: machinev1beta1.MachineHealthCheckSpec{
					MaxUnhealthy: pointerutils.ToPtr(intstr.FromInt32(1)),
					Selector: metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "machine.openshift.io/cluster-api-machine-role",
								Operator: metav1.LabelSelectorOpNotIn,
								Values:   []string{"worker"},
							},
							{
								Key:      "machine.openshift.io/cluster-api-machineset",
								Operator: metav1.LabelSelectorOpExists,
							},
						},
					},
				},
			},
			wantConditions: defaultConditions,
			wantErr:        "",
			assertMHC: func(t *testing.T, ctx context.Context, r *Reconciler) {
				t.Helper()
				mhc := &machinev1beta1.MachineHealthCheck{}
				if err := r.Client.Get(ctx, mhcKey, mhc); err != nil {
					t.Fatalf("expected MHC to exist: %v", err)
				}
				if len(mhc.Spec.Selector.MatchExpressions) != 2 {
					t.Fatalf("expected 2 matchExpressions, got %d", len(mhc.Spec.Selector.MatchExpressions))
				}
				for _, expr := range mhc.Spec.Selector.MatchExpressions {
					if expr.Key == "machine.openshift.io/cluster-api-machine-role" {
						if len(expr.Values) != 1 || expr.Values[0] != "master" {
							t.Errorf("expected values [master], got %v", expr.Values)
						}
						return
					}
				}
				t.Error("required machine-role expression not found")
			},
		},
		{
			name: "Existing MHC with paused annotation is unpaused when not upgrading",
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
			existingMHC: &machinev1beta1.MachineHealthCheck{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-machinehealthcheck",
					Namespace: "openshift-machine-api",
					Annotations: map[string]string{
						MHCPausedAnnotation: "",
					},
				},
				Spec: machinev1beta1.MachineHealthCheckSpec{
					MaxUnhealthy: pointerutils.ToPtr(intstr.FromInt32(1)),
				},
			},
			wantConditions: defaultConditions,
			wantErr:        "",
			assertMHC: func(t *testing.T, ctx context.Context, r *Reconciler) {
				t.Helper()
				mhc := &machinev1beta1.MachineHealthCheck{}
				if err := r.Client.Get(ctx, mhcKey, mhc); err != nil {
					t.Fatalf("expected MHC to exist: %v", err)
				}
				if _, ok := mhc.Annotations[MHCPausedAnnotation]; ok {
					t.Error("expected paused annotation to be removed when not upgrading")
				}
			},
		},
		{
			name: "Existing MHC without paused annotation gets paused during upgrade",
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
			existingMHC: &machinev1beta1.MachineHealthCheck{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aro-machinehealthcheck",
					Namespace: "openshift-machine-api",
				},
				Spec: machinev1beta1.MachineHealthCheckSpec{
					MaxUnhealthy: pointerutils.ToPtr(intstr.FromInt32(1)),
				},
			},
			wantErr: "",
			assertMHC: func(t *testing.T, ctx context.Context, r *Reconciler) {
				t.Helper()
				mhc := &machinev1beta1.MachineHealthCheck{}
				if err := r.Client.Get(ctx, mhcKey, mhc); err != nil {
					t.Fatalf("expected MHC to exist: %v", err)
				}
				if _, ok := mhc.Annotations[MHCPausedAnnotation]; !ok {
					t.Error("expected paused annotation to be added during upgrade")
				}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clientBuilder := testclienthelper.NewAROFakeClientBuilder()
			if tt.instance != nil {
				clientBuilder = clientBuilder.WithObjects(tt.instance)
			}
			if tt.clusterversion == nil {
				clientBuilder = clientBuilder.WithObjects(clusterversionDefault)
			} else {
				clientBuilder = clientBuilder.WithObjects(tt.clusterversion)
			}
			if tt.existingMHC != nil {
				clientBuilder = clientBuilder.WithObjects(tt.existingMHC)
			}

			ctx := context.Background()

			r := NewReconciler(
				logrus.NewEntry(logrus.StandardLogger()),
				clientBuilder.Build(),
			)

			request := ctrl.Request{}
			request.Name = "cluster"

			result, err := r.Reconcile(ctx, request)

			if tt.wantRequeueAfter != result.RequeueAfter {
				t.Errorf("wanted to requeue after %v but was set to %v", tt.wantRequeueAfter, result.RequeueAfter)
			}

			if tt.instance != nil {
				utilconditions.AssertControllerConditions(t, ctx, r.Client, tt.wantConditions)
			}

			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.assertMHC != nil {
				tt.assertMHC(t, ctx, r)
			}
		})
	}
}
