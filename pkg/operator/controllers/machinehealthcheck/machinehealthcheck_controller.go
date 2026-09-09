package machinehealthcheck

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"slices"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/retry"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1 "github.com/openshift/api/config/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const (
	ControllerName      string = "MachineHealthCheck"
	MHCPausedAnnotation string = "cluster.x-k8s.io/paused"

	mhcName      = "aro-machinehealthcheck"
	mhcNamespace = "openshift-machine-api"
)

var requiredMatchExpressions = []metav1.LabelSelectorRequirement{
	{
		Key:      "machine.openshift.io/cluster-api-machine-role",
		Operator: metav1.LabelSelectorOpNotIn,
		Values:   []string{"master"},
	},
	{
		Key:      "machine.openshift.io/cluster-api-machineset",
		Operator: metav1.LabelSelectorOpExists,
	},
}

type Reconciler struct {
	base.AROController
}

func NewReconciler(log *logrus.Entry, client client.Client) *Reconciler {
	return &Reconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   ControllerName,
		},
	}
}

func defaultMachineHealthCheck() *machinev1beta1.MachineHealthCheck {
	return &machinev1beta1.MachineHealthCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mhcName,
			Namespace: mhcNamespace,
		},
		Spec: machinev1beta1.MachineHealthCheckSpec{
			Selector: metav1.LabelSelector{
				MatchExpressions: requiredMatchExpressions,
			},
			UnhealthyConditions: []machinev1beta1.UnhealthyCondition{
				{
					Type:    corev1.NodeReady,
					Status:  corev1.ConditionFalse,
					Timeout: metav1.Duration{Duration: 15 * time.Minute},
				},
				{
					Type:    corev1.NodeReady,
					Status:  corev1.ConditionUnknown,
					Timeout: metav1.Duration{Duration: 15 * time.Minute},
				},
			},
			MaxUnhealthy:       pointerutils.ToPtr(intstr.FromInt32(1)),
			NodeStartupTimeout: &metav1.Duration{Duration: 25 * time.Minute},
		},
	}
}

// Reconcile watches MachineHealthCheck objects, and if any changes,
// reconciles the associated ARO MachineHealthCheck object
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.GetCluster(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.MachineHealthCheckEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")
	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.MachineHealthCheckManaged) {
		err := r.deleteIfExists(ctx, "MachineHealthCheck", &machinev1beta1.MachineHealthCheck{
			ObjectMeta: metav1.ObjectMeta{Name: mhcName, Namespace: mhcNamespace},
		})
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)

			return reconcile.Result{RequeueAfter: time.Hour}, err
		}

		r.ClearConditions(ctx)
		return reconcile.Result{}, nil
	}

	err = r.deleteIfExists(ctx, "PrometheusRule", &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{Name: "mhc-remediation-alert", Namespace: "openshift-machine-api"},
	})
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)

		return reconcile.Result{RequeueAfter: time.Hour}, err
	}

	desired := defaultMachineHealthCheck()

	isUpgrading, err := r.IsClusterUpgrading(ctx)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)

		return reconcile.Result{}, err
	}

	if isUpgrading {
		desired.Annotations = map[string]string{
			MHCPausedAnnotation: "",
		}
	}

	// helps with garbage collection of the resources we are dealing with
	err = controllerutil.SetControllerReference(instance, desired, scheme.Scheme)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.ensureMachineHealthCheck(ctx, desired, isUpgrading)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)

		return reconcile.Result{}, err
	}

	r.ClearConditions(ctx)
	return reconcile.Result{}, nil
}

func (r *Reconciler) ensureMachineHealthCheck(ctx context.Context, desired *machinev1beta1.MachineHealthCheck, isUpgrading bool) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing := &machinev1beta1.MachineHealthCheck{}
		key := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := r.Client.Get(ctx, key, existing)
		if kerrors.IsNotFound(err) {
			r.Log.Infof("creating MachineHealthCheck %s/%s", desired.Namespace, desired.Name)
			return r.Client.Create(ctx, desired)
		}
		if err != nil {
			return err
		}

		patch := existing.DeepCopy()
		changed := false

		// Ensure the controller reference exists so the MHC is garbage-collected with the Cluster CR.
		if metav1.GetControllerOf(patch) == nil && len(desired.OwnerReferences) > 0 {
			patch.OwnerReferences = append(patch.OwnerReferences, desired.OwnerReferences[0])
			changed = true
		}
		// OpenShift machine-api-controllers defaults maxUnhealthy to "100%" when unset,
		// so nil is not expected. Default a zero value back to 1.
		if patch.Spec.MaxUnhealthy != nil &&
			((patch.Spec.MaxUnhealthy.Type == intstr.Int && patch.Spec.MaxUnhealthy.IntVal == 0) ||
				(patch.Spec.MaxUnhealthy.Type == intstr.String && patch.Spec.MaxUnhealthy.StrVal == "0")) {
			patch.Spec.MaxUnhealthy = pointerutils.ToPtr(intstr.FromInt32(1))
			changed = true
		}

		if ensureRequiredSelectors(&patch.Spec.Selector) {
			changed = true
		}

		_, paused := patch.Annotations[MHCPausedAnnotation]
		if isUpgrading && !paused {
			if patch.Annotations == nil {
				patch.Annotations = map[string]string{}
			}
			patch.Annotations[MHCPausedAnnotation] = ""
			changed = true
		} else if !isUpgrading && paused {
			delete(patch.Annotations, MHCPausedAnnotation)
			changed = true
		}

		if !changed {
			return nil
		}
		r.Log.Infof("updating MachineHealthCheck %s/%s", patch.Namespace, patch.Name)
		return r.Client.Update(ctx, patch)
	})
}

func (r *Reconciler) deleteIfExists(ctx context.Context, kind string, obj client.Object) error {
	err := r.Client.Delete(ctx, obj)
	if kerrors.IsNotFound(err) {
		return nil
	}
	if err == nil {
		r.Log.Infof("deleted %s %s/%s", kind, obj.GetNamespace(), obj.GetName())
	}
	return err
}

func ensureRequiredSelectors(selector *metav1.LabelSelector) bool {
	changed := false

	for _, req := range requiredMatchExpressions {
		// Remove any requirements for the same key with a different operator (tampered selector).
		for i := 0; i < len(selector.MatchExpressions); {
			e := selector.MatchExpressions[i]
			if e.Key == req.Key && e.Operator != req.Operator {
				selector.MatchExpressions = append(selector.MatchExpressions[:i], selector.MatchExpressions[i+1:]...)
				changed = true
				continue
			}
			i++
		}

		if idx := findMatchExpression(selector.MatchExpressions, req.Key, req.Operator); idx >= 0 {
			if !slices.Equal(selector.MatchExpressions[idx].Values, req.Values) {
				selector.MatchExpressions[idx].Values = slices.Clone(req.Values)
				changed = true
			}
			continue
		}

		reqCopy := req
		reqCopy.Values = slices.Clone(req.Values)
		selector.MatchExpressions = append(selector.MatchExpressions, reqCopy)
		changed = true
	}

	return changed
}

func findMatchExpression(exprs []metav1.LabelSelectorRequirement, key string, op metav1.LabelSelectorOperator) int {
	for i, e := range exprs {
		if e.Key == key && e.Operator == op {
			return i
		}
	}
	return -1
}

// SetupWithManager will manage only our MHC resource with our specific controller name
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Named(ControllerName).
		Owns(&machinev1beta1.MachineHealthCheck{}).
		Owns(&monitoringv1.PrometheusRule{}).
		Watches(
			&configv1.ClusterVersion{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicates.ClusterVersion),
		).
		Complete(r)
}
