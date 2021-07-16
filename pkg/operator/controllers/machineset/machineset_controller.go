package machineset

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"regexp"

	"github.com/Azure/go-autorest/autorest/to"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
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

type Reconciler struct {
	log *logrus.Entry

	arocli aroclient.Interface
	maocli maoclient.Interface
}

// MachineSet reconciler watches MachineSet objects for changes, evaluates total worker replica count, and reverts changes if needed.
func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, maocli maoclient.Interface) *Reconciler {
	return &Reconciler{
		log:    log,
		arocli: arocli,
		maocli: maocli,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.Features.ReconcileMachineSet {
		return reconcile.Result{}, nil
	}

	machinesets, err := r.maocli.MachineV1beta1().MachineSets(machineSetsNamespace).List(ctx, metav1.ListOptions{LabelSelector: "machine.openshift.io/cluster-api-machine-role=worker"})
	if err != nil {
		return reconcile.Result{}, err
	}

	aroMachineset, err := r.maocli.MachineV1beta1().MachineSets(machineSetsNamespace).Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Count amount of total current worker replicas (including custom MachineSets, if present)
	replicaCount := 0
	for _, machineset := range machinesets.Items {
		if machineset.Spec.Replicas != nil {
			replicaCount += int(*machineset.Spec.Replicas)
		}
	}

	// Verify that the MachineSet that was modified matches the cluster's InfraID, otherwise ignore it
	matches, err := regexp.Match(instance.Spec.InfraID, []byte(aroMachineset.ObjectMeta.Name))
	if err != nil {
		r.log.Error(err)
	}

	if replicaCount < 3 && matches {
		r.log.Info("Found less than 3 worker replicas. The MachineSet controller will attempt scaling.")
		aroMachineset.Spec.Replicas = to.Int32Ptr(int32(3-replicaCount) + *aroMachineset.Spec.Replicas) // Add replicas to the object, and call Update
		_, err := r.maocli.MachineV1beta1().MachineSets(machineSetsNamespace).Update(ctx, aroMachineset, metav1.UpdateOptions{})
		if err != nil {
			r.log.Errorf("Error updating MachineSet '%v': %v", aroMachineset, err)
		}
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	machineSetPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetNamespace() == machineSetsNamespace
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&machinev1beta1.MachineSet{}, builder.WithPredicates(machineSetPredicate)).
		Watches(&source.Kind{Type: &machinev1beta1.MachineSet{}}, &handler.EnqueueRequestForObject{}). // Reconcile on changes to MachineSet objects
		Named(controllers.MachineSetControllerName).
		Complete(r)
}
