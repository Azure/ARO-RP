package etchosts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	ClusterControllerName = "EtcHostsCluster"
)

var (
	etchostsMasterMCMetadata = &mcv1.MachineConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "99-master-aro-etc-hosts-gateway-domains",
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "MachineConfig",
		},
	}
	etchostsWorkerMCMetadata = &mcv1.MachineConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "99-worker-aro-etc-hosts-gateway-domains",
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "MachineConfig",
		},
	}
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

	allowReconcile, err := r.AllowRebootCausingReconciliation(ctx, instance)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	// EtchostsManaged = false, remove machine configs
	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.EtcHostsManaged) && allowReconcile {
		r.Log.Debug("etchosts managed is false, removing machine configs")
		err = r.removeMachineConfig(ctx, etchostsMasterMCMetadata)
		if kerrors.IsNotFound(err) {
			r.ClearDegraded(ctx)
			return reconcile.Result{}, nil
		}
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)
			return reconcile.Result{}, err
		}

		err = r.removeMachineConfig(ctx, etchostsWorkerMCMetadata)
		if kerrors.IsNotFound(err) {
			r.ClearDegraded(ctx)
			return reconcile.Result{}, nil
		}
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)
			return reconcile.Result{}, err
		}

		r.ClearConditions(ctx)
		r.Log.Debug("etchosts managed is false, machine configs removed")
		return reconcile.Result{}, nil
	}

	// EtchostsManaged = true, create machine configs if missing
	r.Log.Debug("running")
	// If 99-master-aro-etc-hosts-gateway-domains doesn't exist, create it
	mcp := &mcv1.MachineConfigPool{}
	mc := &mcv1.MachineConfig{}

	err = r.Client.Get(ctx, types.NamespacedName{Name: "master"}, mcp)
	if kerrors.IsNotFound(err) {
		r.Log.Debug(err)
		r.ClearDegraded(ctx)
		return reconcile.Result{}, nil
	}
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}
	if mcp.GetDeletionTimestamp() != nil {
		return reconcile.Result{}, nil
	}

	err = r.Client.Get(ctx, types.NamespacedName{Name: "99-master-aro-etc-hosts-gateway-domains"}, mc)
	if kerrors.IsNotFound(err) {
		r.Log.Debug("99-master-aro-etc-hosts-gateway-domains not found, creating it")
		err = reconcileMachineConfigs(ctx, instance, "master", r.dh, allowReconcile, *mcp)
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)
			return reconcile.Result{}, err
		}
		r.ClearDegraded(ctx)
		return reconcile.Result{Requeue: true}, nil
	}
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	// If 99-worker-aro-etc-hosts-gateway-domains doesn't exist, create it
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
	if mcp.GetDeletionTimestamp() != nil {
		return reconcile.Result{}, nil
	}

	err = r.Client.Get(ctx, types.NamespacedName{Name: "99-worker-aro-etc-hosts-gateway-domains"}, mc)
	if kerrors.IsNotFound(err) {
		r.Log.Debug("99-worker-aro-etc-hosts-gateway-domains not found, creating it")
		r.ClearDegraded(ctx)
		err = reconcileMachineConfigs(ctx, instance, "worker", r.dh, allowReconcile, *mcp)
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	r.ClearConditions(ctx)
	return reconcile.Result{}, nil
}

// SetupWithManager setup our mananger to watch for changes to MCP and ARO Cluster obj
func (r *EtcHostsClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log.Info("starting etchosts-cluster controller")

	etcHostsBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Watches(&mcv1.MachineConfigPool{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&mcv1.MachineConfig{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{}))

	return etcHostsBuilder.
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}, predicate.LabelChangedPredicate{})).
		Named(ClusterControllerName).
		Complete(r)
}

func (r *EtcHostsClusterReconciler) removeMachineConfig(ctx context.Context, mc *mcv1.MachineConfig) error {
	r.Log.Debugf("removing machine config %s", mc.Name)
	err := r.Client.Delete(ctx, mc)
	return err
}
