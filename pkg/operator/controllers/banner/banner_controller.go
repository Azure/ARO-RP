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
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

// BannerReconciler is the controller struct
type BannerReconciler struct {
	arocli     aroclient.Interface
	log        *logrus.Entry
	consolecli consoleclient.Interface
}

// NewReconciler creates a new Reconciler
func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, consolecli consoleclient.Interface) *BannerReconciler {
	return &BannerReconciler{
		arocli:     arocli,
		log:        log,
		consolecli: consolecli,
	}
}

// Reconcile posts or removes the notification banner
func (r *BannerReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.Features.ReconcileBanner {
		// reconciling Banners is disabled
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, r.reconcileBanner(ctx, instance)
}

// SetupWithManager creates the controller
func (r *BannerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Watches(&source.Kind{Type: &consolev1.ConsoleNotification{}}, &handler.EnqueueRequestForObject{}). //watching ConsoleNotifications in case a user edits it
		Named(controllers.BannerControllerName).
		Complete(r)
}
