package etchosts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"cmp"
	"context"
	"slices"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1 "github.com/openshift/api/config/v1"
	mcv1 "github.com/openshift/api/machineconfiguration/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
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
	ch clienthelper.Interface
}

func NewClusterReconciler(log *logrus.Entry, client client.Client, ch clienthelper.Interface) *EtcHostsClusterReconciler {
	return &EtcHostsClusterReconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   ClusterControllerName,
		},
		ch: ch,
	}
}

// Reconcile watches ARO EtcHosts MachineConfig objects, and if any changes, reconciles it
func (r *EtcHostsClusterReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
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

	// If we do not allow reconciliation right now, return.
	//
	// NOTE: Generally we would want to unconditionally create the
	// MachineConfigs if they are missing, but since the etchosts functionality
	// was rolled out when there are a substantial number of clusters, we don't
	// want the flicking of this switch on older clusters to cause instant
	// reboots. Hence, wait until we are allowed to reboot (mostly, before
	// upgrades) if this is changed.
	if !allowReconcile {
		r.Log.Debug("reboot-causing reconciliation not allowed right now")
		r.ClearConditions(ctx)
		return reconcile.Result{}, nil
	}

	// EtchostsManaged = false, remove machine configs
	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.EtcHostsManaged) {
		r.Log.Debug("etchosts managed is false, removing machine configs")

		continueToken := ""
		for {
			mcs := &mcv1.MachineConfigList{}
			err := r.ch.List(ctx, mcs, client.Continue(continueToken))
			if err != nil {
				r.Log.Error(err)
				r.SetDegraded(ctx, err)
				return reconcile.Result{}, err
			}

			for _, mc := range mcs.Items {
				// Filter down to our named etchosts machineconfigs
				if etcHostsRegex.FindStringSubmatch(mc.Name) != nil {
					err = r.ch.Delete(ctx, &mc)
					if err != nil {
						r.Log.Error(err)
						r.SetDegraded(ctx, err)
						return reconcile.Result{}, err
					}
				}
			}

			continueToken = mcs.Continue
			if continueToken == "" {
				break
			}
		}

		r.ClearConditions(ctx)
		r.Log.Debug("etchosts managed is false, machine configs removed")
		return reconcile.Result{}, nil
	}

	// EtchostsManaged = true, create machine configs for all MCPs if missing
	r.Log.Debug("running")

	pools := &mcv1.MachineConfigPoolList{}
	err = r.ch.List(ctx, pools)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	// Sort for test
	slices.SortStableFunc(pools.Items, func(a, b mcv1.MachineConfigPool) int {
		return cmp.Compare(a.Name, b.Name)
	})
	err = reconcileMachineConfigs(ctx, instance, r.ch, allowReconcile, pools.Items...)
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

	clusterVersionPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == "version"
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Named(ClusterControllerName).
		Watches(
			&configv1.ClusterVersion{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(clusterVersionPredicate),
		).
		Complete(r)
}
