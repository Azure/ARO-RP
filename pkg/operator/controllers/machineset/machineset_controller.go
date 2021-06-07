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

type MachineSetReconciler struct {
	log    *logrus.Entry
	maocli maoclient.Interface
	arocli aroclient.Interface
}

func NewMachineSetReconciler(log *logrus.Entry, maocli maoclient.Interface, arocli aroclient.Interface) *MachineSetReconciler {
	return &MachineSetReconciler{
		log:    log,
		maocli: maocli,
		arocli: arocli,
	}
}

func (r *MachineSetReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.Features.ReconcileMachineSet {
		return reconcile.Result{}, nil
	}

	machinesetObject, err := r.maocli.MachineV1beta1().MachineSets(machineSetsNamespace).Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Count amount of current worker replicas
	replicaCount := 0
	machinesets, err := r.maocli.MachineV1beta1().MachineSets(machineSetsNamespace).List(ctx, metav1.ListOptions{LabelSelector: "machine.openshift.io/cluster-api-machine-role=worker"})
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, machineset := range machinesets.Items {
		if machineset.Spec.Replicas != nil {
			replicaCount += int(*machineset.Spec.Replicas)
			r.log.Infof("Worker count is %v for InfraID '%v'", replicaCount, instance.Spec.InfraID)
		}
	}

	// Verify that MachineSet InfraID matches cluster InfraID
	matches, err := regexp.Match(instance.Spec.InfraID, []byte(machinesetObject.ObjectMeta.Name))
	if err != nil {
		r.log.Error(err)
	}

	// Scale up
	if replicaCount < 3 && matches {
		r.log.Error("Found less than 3 worker replicas. The MachineSet controller will attempt scaling.")
		machinesetObject.Spec.Replicas = to.Int32Ptr(int32(1) + *machinesetObject.Spec.Replicas)
		_, err := r.maocli.MachineV1beta1().MachineSets(machineSetsNamespace).Update(ctx, machinesetObject, metav1.UpdateOptions{})
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, err
}

func (r *MachineSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	machineSetPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetNamespace() == machineSetsNamespace
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&machinev1beta1.MachineSet{}, builder.WithPredicates(machineSetPredicate)).
		Watches(&source.Kind{Type: &machinev1beta1.Machine{}}, &handler.EnqueueRequestForObject{}). // Reconcile on changes to MachineSet objects
		Named(controllers.MachineSetControllerName).
		Complete(r)
}
