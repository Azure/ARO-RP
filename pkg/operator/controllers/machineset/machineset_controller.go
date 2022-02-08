package machineset

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

const (
	CONFIG_NAMESPACE string = "aro.machineset"
	ENABLED          string = CONFIG_NAMESPACE + ".enabled"
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

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(ENABLED) {
		// controller is disabled
		return reconcile.Result{}, nil
	}

	modifiedMachineset, err := r.maocli.MachineV1beta1().MachineSets(machineSetsNamespace).Get(ctx, request.Name, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return reconcile.Result{}, err
	}
	if err != nil {
		return reconcile.Result{}, err
	}

	machinesets, err := r.maocli.MachineV1beta1().MachineSets(machineSetsNamespace).List(ctx, metav1.ListOptions{LabelSelector: "machine.openshift.io/cluster-api-machine-role=worker"})
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
		_, err := r.maocli.MachineV1beta1().MachineSets(machineSetsNamespace).Update(ctx, modifiedMachineset, metav1.UpdateOptions{})
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	machineSetPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		role := o.GetLabels()["machine.openshift.io/cluster-api-machine-role"]
		return strings.EqualFold("worker", role)
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&machinev1beta1.MachineSet{}, builder.WithPredicates(machineSetPredicate)).
		Named(controllers.MachineSetControllerName).
		Complete(r)
}
