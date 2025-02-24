package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	ControllerName = "Workaround"
)

// Reconciler the point of the workaround controller is to apply
// workarounds that we have unitl upstream fixes are available.
type Reconciler struct {
	log *logrus.Entry

	workarounds []Workaround

	client client.Client
}

func NewReconciler(log *logrus.Entry, client client.Client) *Reconciler {
	return &Reconciler{
		log:         log,
		workarounds: []Workaround{NewSystemReserved(log, client)},
		client:      client,
	}
}

// Reconcile makes sure that the workarounds are applied or removed as per the OpenShift version.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.WorkaroundEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")

	cv := &configv1.ClusterVersion{}
	err = r.client.Get(ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		return reconcile.Result{}, err
	}

	clusterVersion, err := version.GetClusterVersion(cv)
	if err != nil {
		r.log.Errorf("error getting the OpenShift version: %v", err)
		return reconcile.Result{}, err
	}

	for _, wa := range r.workarounds {
		if wa.IsRequired(clusterVersion, instance) {
			err = wa.Ensure(ctx)
		} else {
			err = wa.Remove(ctx)
		}

		if err != nil {
			r.log.Errorf("workaround %s returned error %v", wa.Name(), err)
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{RequeueAfter: time.Hour}, nil
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Watches(&configv1.ClusterVersion{}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(predicates.ClusterVersion)).
		Named(ControllerName).
		Complete(r)
}
