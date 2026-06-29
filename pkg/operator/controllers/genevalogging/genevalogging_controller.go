package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	securityv1 "github.com/openshift/api/security/v1"

	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	ControllerName = "GenevaLogging"

	// full pullspec of otel exporter image
	controllerOTelPullSpec = "aro.genevalogging.otel.pullSpec"
)

// Reconciler reconciles a Cluster object
type Reconciler struct {
	base.AROController

	dh dynamichelper.Interface
}

func NewReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *Reconciler {
	return &Reconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   ControllerName,
		},
		dh: dh,
	}
}

func (r *Reconciler) ensureResources(ctx context.Context, instance *arov1alpha1.Cluster) error {
	if err := r.cleanupStaleResources(ctx); err != nil {
		return err
	}

	resources, err := r.resources(ctx, instance)
	if err != nil {
		return err
	}

	err = dynamichelper.SetControllerReferences(resources, instance)
	if err != nil {
		return err
	}

	err = dynamichelper.Prepare(resources)
	if err != nil {
		return err
	}

	err = r.dh.Ensure(ctx, resources...)
	if err != nil {
		return err
	}

	// OTel daemonsets should never be manually "scaled down" via pod template
	// node selectors. Reconciliation owns this field and clears any drift.
	if err := r.clearOTelDaemonSetNodeSelectors(ctx); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) clearOTelDaemonSetNodeSelectors(ctx context.Context) error {
	for _, name := range []string{MasterDaemonsetName, WorkerDaemonsetName} {
		ds := &appsv1.DaemonSet{}
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: kubeNamespace, Name: name}, ds)
		if kerrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}

		if len(ds.Spec.Template.Spec.NodeSelector) == 0 {
			continue
		}

		ds.Spec.Template.Spec.NodeSelector = nil
		if err := r.Client.Update(ctx, ds); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) cleanupStaleResources(ctx context.Context) error {
	type staleResource struct {
		groupKind string
		namespace string
		name      string
	}

	stale := []staleResource{
		{"DaemonSet.apps", kubeNamespace, "mdsd"},
		{"ConfigMap", kubeNamespace, "fluent-config"},
		{"Secret", kubeNamespace, "certificates"},
		{"ConfigMap", kubeNamespace, legacyGatewayCACMName},
	}

	for _, res := range stale {
		if err := r.dh.EnsureDeleted(ctx, res.groupKind, res.namespace, res.name); err != nil {
			return err
		}
	}

	return nil
}

// Reconcile the genevalogging deployment.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.GetCluster(ctx)
	if err != nil {
		r.Log.Error(err)
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(pkgoperator.GenevaLoggingEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")
	err = r.ensureResources(ctx, instance)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	r.ClearConditions(ctx)
	return reconcile.Result{}, nil
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Namespace{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&monitoringv1.PodMonitor{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&securityv1.SecurityContextConstraints{}).
		Named(ControllerName).
		Complete(r)
}
