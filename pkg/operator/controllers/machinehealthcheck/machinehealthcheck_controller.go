package machinehealthcheck

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"slices"
	"time"

	_ "embed"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"

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

//go:embed staticresources/machinehealthcheck.yaml
var machinehealthcheckYaml []byte

const (
	ControllerName      string = "MachineHealthCheck"
	MHCPausedAnnotation string = "cluster.x-k8s.io/paused"
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
			ObjectMeta: metav1.ObjectMeta{Name: "aro-machinehealthcheck", Namespace: "openshift-machine-api"},
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

	resource, _, err := scheme.Codecs.UniversalDeserializer().Decode(machinehealthcheckYaml, nil, nil)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)

		return reconcile.Result{}, err
	}

	desired, ok := resource.(*machinev1beta1.MachineHealthCheck)
	if !ok {
		r.Log.Error("decoded resource is not a MachineHealthCheck")
		return reconcile.Result{}, nil
	}

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
		if idx := findMatchExpression(selector.MatchExpressions, req.Key, req.Operator); idx >= 0 {
			if !slices.Equal(selector.MatchExpressions[idx].Values, req.Values) {
				selector.MatchExpressions[idx].Values = req.Values
				changed = true
			}
		} else {
			selector.MatchExpressions = append(selector.MatchExpressions, req)
			changed = true
		}
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
