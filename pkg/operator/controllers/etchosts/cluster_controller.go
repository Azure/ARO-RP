package etchosts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	ClusterControllerName = "EtcHostsCluster"
)

type EtcHostsClusterReconciler struct {
	base.AROController

	dh dynamichelper.Interface
}

func NewClusterReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *EtcHostsClusterReconciler {
	return &EtcHostsClusterReconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   ClusterControllerName,
		},
		dh: dh,
	}
}

// Reconcile watches ARO EtcHosts MachineConfig objects, and if any changes, reconciles it
func (r *EtcHostsClusterReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.Log.Debugf("reconcile MachineConfig openshift-machine-api/%s", request.Name)

	instance, err := r.GetCluster(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.EtcHostsEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")

	mc := &mcv1.MachineConfig{}
	mcp := &mcv1.MachineConfigPool{}

	// If 99-master-aro-etc-hosts-gateway-domains doesn't exist, create it
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: "openshift-machine-api", Name: "99-master-aro-etc-hosts-gateway-domains"}, mc)
	if kerrors.IsNotFound(err) {
		err = r.Client.Get(ctx, types.NamespacedName{Name: "master"}, mcp)
		if kerrors.IsNotFound(err) {
			r.ClearDegraded(ctx)
			return reconcile.Result{}, nil
		}
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)
			return reconcile.Result{}, err
		}
		err = reconcileMachineConfigs(ctx, instance, "master", r.dh, *mcp)
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)
			return reconcile.Result{}, err
		}
	}

	// If 99-worker-aro-etc-hosts-gateway-domains doesn't exist, create it
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: "openshift-machine-api", Name: "99-worker-aro-etc-hosts-gateway-domains"}, mc)
	if kerrors.IsNotFound(err) {
		err = r.Client.Get(ctx, types.NamespacedName{Name: "worker"}, mcp)
		if kerrors.IsNotFound(err) {
			r.ClearDegraded(ctx)
			return reconcile.Result{}, nil
		}
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)
			return reconcile.Result{}, err
		}
		err = reconcileMachineConfigs(ctx, instance, "worker", r.dh, *mcp)
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)
			return reconcile.Result{}, err
		}
	}

	r.ClearConditions(ctx)
	return reconcile.Result{}, nil
}

// SetupWithManager setup our mananger to watch for changes to MCP and ARO Cluster obj
func (r *EtcHostsClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log.Info("starting etchosts-cluster controller")

	etcHostsBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Watches(&source.Kind{Type: &mcv1.MachineConfigPool{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&source.Kind{Type: &mcv1.MachineConfig{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{}))

	return etcHostsBuilder.
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}, predicate.LabelChangedPredicate{})).
		Named(ClusterControllerName).
		Complete(r)
}
