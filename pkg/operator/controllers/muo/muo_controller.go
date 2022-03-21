package muo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	ControllerName = "ManagedUpgradeOperator"

	controllerEnabled  = "rh.srep.muo.enabled"
	controllerManaged  = "rh.srep.muo.managed"
	controllerPullSpec = "rh.srep.muo.deploy.pullspec"
)

type Reconciler struct {
	arocli   aroclient.Interface
	deployer Deployer

	readinessPollTime time.Duration
	readinessTimeout  time.Duration
}

func NewReconciler(arocli aroclient.Interface, kubernetescli kubernetes.Interface, dh dynamichelper.Interface) *Reconciler {
	return &Reconciler{
		arocli:   arocli,
		deployer: newDeployer(kubernetescli, dh),

		readinessPollTime: 10 * time.Second,
		readinessTimeout:  5 * time.Minute,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		// controller is disabled
		return reconcile.Result{}, nil
	}

	managed := instance.Spec.OperatorFlags.GetWithDefault(controllerManaged, "")

	// If enabled and managed=true, install MUO
	// If enabled and managed=false, remove the MUO deployment
	// If enabled and managed is missing, do nothing
	if strings.EqualFold(managed, "true") {
		err = r.deployer.CreateOrUpdate(ctx, instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Check that MUO has become ready, wait up to readinessTimeout (default 5min)
		timeoutCtx, cancel := context.WithTimeout(ctx, r.readinessTimeout)
		defer cancel()

		err := wait.PollImmediateUntil(r.readinessPollTime, func() (bool, error) {
			return r.deployer.IsReady(ctx)
		}, timeoutCtx.Done())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("Managed Upgrade Operator deployment timed out on Ready: %w", err)
		}
	} else if strings.EqualFold(managed, "false") {
		err := r.deployer.Remove(ctx)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	builder := ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate))

	resources, err := r.deployer.Resources("")
	if err != nil {
		return err
	}

	for _, i := range resources {
		o, ok := i.(client.Object)
		if ok {
			builder.Owns(o)
		}
	}

	return builder.
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}, predicate.LabelChangedPredicate{})).
		Named(ControllerName).
		Complete(r)
}
