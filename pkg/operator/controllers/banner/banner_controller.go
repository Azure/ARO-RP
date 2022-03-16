package banner

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	consolev1 "github.com/openshift/api/console/v1"
	consoleclient "github.com/openshift/client-go/console/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
)

const (
	ControllerName = "Banner"

	controllerEnabled = "aro.banner.enabled"
)

// BannerReconciler is the controller struct
type Reconciler struct {
	log *logrus.Entry

	arocli     aroclient.Interface
	consolecli consoleclient.Interface
}

// NewReconciler creates a new Reconciler
func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, consolecli consoleclient.Interface) *Reconciler {
	return &Reconciler{
		log:        log,
		arocli:     arocli,
		consolecli: consolecli,
	}
}

// Reconcile posts or removes the notification banner
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		// controller is disabled
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, r.reconcileBanner(ctx, instance)
}

// SetupWithManager creates the controller
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	aroBannerPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == BannerName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		// watching ConsoleNotifications in case a user edits it
		Watches(&source.Kind{Type: &consolev1.ConsoleNotification{}}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(aroBannerPredicate)).
		Named(ControllerName).
		Complete(r)
}
