package cpms

import (
	"context"

	"github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	machinev1 "github.com/openshift/api/machine/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	ControllerName         = "CPMSController"
	SingletonCPMSName      = "cluster"
	SingletonCPMSNamespace = "openshift-machine-api"
)

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

// CPMS reconciler will disable the cluster CPMS if `aro.cpms.enabled` is false or missing.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.GetCluster(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if instance.Spec.OperatorFlags.GetSimpleBoolean(operator.CPMSEnabled) {
		r.Log.Infof("Flag %s is true, will not deactivate CPMS", operator.CPMSEnabled)
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")
	cpms := &machinev1.ControlPlaneMachineSet{}
	err = r.Client.Get(
		ctx,
		types.NamespacedName{Name: SingletonCPMSName, Namespace: SingletonCPMSNamespace},
		cpms,
	)
	if err != nil {
		if kerrors.IsNotFound(err) {
			r.Log.Info("CPMS is not present on cluster, nothing to do")
			return ctrl.Result{}, nil
		}
		r.Log.Errorf("Error when retrieving CPMS: %v", err)
		return ctrl.Result{}, err
	}

	if cpms.Spec.State == machinev1.ControlPlaneMachineSetStateInactive {
		r.Log.Info("CPMS is inactive, nothing to do")
		return ctrl.Result{}, nil
	}

	// disable CPMS by deleting it
	// https://docs.openshift.com/container-platform/4.12/machine_management/control_plane_machine_management/cpmso-disabling.html
	return ctrl.Result{}, r.Client.Delete(ctx, cpms)
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log.Info("starting cpms controller")

	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(aroClusterPredicate, predicate.GenerationChangedPredicate{}))).
		Watches(
			&machinev1.ControlPlaneMachineSet{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{}), // only watch for spec changes
		).
		Named(ControllerName).
		Complete(r)
}
