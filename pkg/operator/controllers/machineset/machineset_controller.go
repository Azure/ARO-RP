package machineset

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
)

const (
	ControllerName = "MachineSet"
)

type Reconciler struct {
	base.AROController
}

// MachineSet reconciler watches MachineSet objects for changes, evaluates total worker replica count, and reverts changes if needed.
func NewReconciler(log *logrus.Entry, client client.Client) *Reconciler {
	return &Reconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   ControllerName,
		},
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.GetCluster(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.MachineSetEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")

	modifiedMachineset := &machinev1beta1.MachineSet{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: machineSetsNamespace}, modifiedMachineset)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)

		return reconcile.Result{}, err
	}

	machinesets := &machinev1beta1.MachineSetList{}
	selector, _ := labels.Parse("machine.openshift.io/cluster-api-machine-role=worker")
	err = r.Client.List(ctx, machinesets, &client.ListOptions{
		Namespace:     machineSetsNamespace,
		LabelSelector: selector,
	})
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)

		return reconcile.Result{}, err
	}

	// Count amount of total current worker replicas
	replicaCount := 0
	for _, machineset := range machinesets.Items {
		// If there are any custom machinesets in the list, bail and don't requeue
		if !strings.Contains(machineset.Name, instance.Spec.InfraID) {
			r.ClearDegraded(ctx)

			return reconcile.Result{}, nil
		}
		if machineset.Spec.Replicas != nil {
			replicaCount += int(*machineset.Spec.Replicas)
		}
	}

	if replicaCount < minSupportedReplicas {
		r.Log.Infof("Found less than %v worker replicas. The MachineSet controller will attempt scaling.", minSupportedReplicas)
		// Add replicas to the object, and call Update
		modifiedMachineset.Spec.Replicas = to.Int32Ptr(int32(minSupportedReplicas-replicaCount) + *modifiedMachineset.Spec.Replicas)
		err := r.Client.Update(ctx, modifiedMachineset)
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)

			return reconcile.Result{}, err
		}
	}

	r.ClearConditions(ctx)
	return reconcile.Result{}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&machinev1beta1.MachineSet{}, builder.WithPredicates(predicates.MachineRoleWorker)).
		Named(ControllerName).
		Complete(r)
}
