package banner

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	consolev1 "github.com/openshift/api/console/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
)

const (
	ControllerName = "Banner"
)

// BannerReconciler is the controller struct
type Reconciler struct {
	log *logrus.Entry

	client client.Client
}

// NewReconciler creates a new Reconciler
func NewReconciler(log *logrus.Entry, client client.Client) *Reconciler {
	return &Reconciler{
		log:    log,
		client: client,
	}
}

// Reconcile posts or removes the notification banner
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.BannerEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")
	return reconcile.Result{}, r.reconcileBanner(ctx, instance)
}

// SetupWithManager creates the controller
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroBannerPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == BannerName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		// watching ConsoleNotifications in case a user edits it
		Watches(&consolev1.ConsoleNotification{}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(aroBannerPredicate)).
		Named(ControllerName).
		Complete(r)
}
