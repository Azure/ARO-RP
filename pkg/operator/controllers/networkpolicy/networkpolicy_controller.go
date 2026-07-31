package networkpolicy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	_ "embed"

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
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

//go:embed staticresources/adminnetworkpolicy.yaml
var adminNetworkPolicyYaml []byte

const ControllerName = "NetworkPolicy"

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

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.GetCluster(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.CCOPprofNetworkPolicyEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	lt417, err := r.clusterVersionLessThan(ctx, "4.17.0")
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}
	if lt417 {
		r.Log.Debug("cluster version below 4.17, AdminNetworkPolicy not available")
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")

	uns, err := dynamichelper.DecodeUnstructured(adminNetworkPolicyYaml)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	err = r.dh.Ensure(ctx, uns)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	r.ClearConditions(ctx)
	return reconcile.Result{}, nil
}

func (r *Reconciler) clusterVersionLessThan(ctx context.Context, target string) (bool, error) {
	cv := &configv1.ClusterVersion{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: "version"}, cv); err != nil {
		return false, err
	}
	clusterVersion, err := version.GetClusterVersion(cv)
	if err != nil {
		return false, fmt.Errorf("getting cluster version: %w", err)
	}
	ver, err := version.ParseVersion(target)
	if err != nil {
		return false, fmt.Errorf("parsing target version %s: %w", target, err)
	}
	return clusterVersion.Lt(ver), nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Watches(
			&configv1.ClusterVersion{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicates.ClusterVersion),
		).
		Named(ControllerName).
		Complete(r)
}
