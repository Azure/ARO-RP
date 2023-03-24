package machineset

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

const (
	ControllerName = "MachineSet"

	ControllerEnabled = "aro.machineset.enabled"
)

type Reconciler struct {
	log *logrus.Entry

	client client.Client
}

// MachineSet reconciler watches MachineSet objects for changes, evaluates total worker replica count, and reverts changes if needed.
func NewReconciler(log *logrus.Entry, client client.Client) *Reconciler {
	return &Reconciler{
		log:    log,
		client: client,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(ControllerEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")
	modifiedMachineset := &machinev1beta1.MachineSet{}
	err = r.client.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: machineSetsNamespace}, modifiedMachineset)
	if err != nil {
		return reconcile.Result{}, err
	}

	machinesets := &machinev1beta1.MachineSetList{}
	selector, _ := labels.Parse("machine.openshift.io/cluster-api-machine-role=worker")
	err = r.client.List(ctx, machinesets, &client.ListOptions{
		Namespace:     machineSetsNamespace,
		LabelSelector: selector,
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Count amount of total current worker replicas
	replicaCount := 0
	for _, machineset := range machinesets.Items {
		// If there are any custom machinesets in the list, bail and don't requeue
		if !strings.Contains(machineset.Name, instance.Spec.InfraID) {
			return reconcile.Result{}, nil
		}
		if machineset.Spec.Replicas != nil {
			replicaCount += int(*machineset.Spec.Replicas)
		}
	}

	if replicaCount < minSupportedReplicas {
		r.log.Infof("Found less than %v worker replicas. The MachineSet controller will attempt scaling.", minSupportedReplicas)
		// Add replicas to the object, and call Update
		modifiedMachineset.Spec.Replicas = to.Int32Ptr(int32(minSupportedReplicas-replicaCount) + *modifiedMachineset.Spec.Replicas)
		err := r.client.Update(ctx, modifiedMachineset)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	machineSetPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		role := o.GetLabels()["machine.openshift.io/cluster-api-machine-role"]
		return strings.EqualFold("worker", role)
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&machinev1beta1.MachineSet{}, builder.WithPredicates(machineSetPredicate)).
		Named(ControllerName).
		Complete(r)
}
