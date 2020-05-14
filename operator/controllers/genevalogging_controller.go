package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aro "github.com/Azure/ARO-RP/operator/api/v1alpha1"
	arov1alpha1 "github.com/Azure/ARO-RP/operator/api/v1alpha1"
)

// GenevaloggingReconciler reconciles a Cluster object
type GenevaloggingReconciler struct {
	client.Client
	Log    *logrus.Entry
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

func (r *GenevaloggingReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	operatorNs, err := OperatorNamespace()
	if err != nil {
		return ReconcileResultError, err
	}

	if request.Name != arov1alpha1.SingletonClusterName || request.Namespace != operatorNs {
		return ReconcileResultIgnore, nil
	}
	r.Log.Info("Reconsiling genevalogging deployment")

	ctx := context.TODO()
	instance := &aro.Cluster{}
	err = r.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		// Error reading the object or not found - requeue the request.
		return ReconcileResultError, err
	}

	if instance.Spec.ResourceID == "" {
		r.Log.Info("Skipping as ClusterSpec not set")
		return ReconcileResultRequeue, nil
	}
	err = r.reconsileGenevaLogging(ctx, instance)
	if err != nil {
		r.Log.Error(err, "reconsileGenevaLogging")
		return ReconcileResultError, err
	}

	r.Log.Info("done, requeueing")
	return ReconcileResultRequeue, nil
}

func (r *GenevaloggingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		Complete(r)
}
