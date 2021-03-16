package azurensg

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

//AzureNSGReconciler is the controller struct
type AzureNSGReconciler struct {
	arocli aroclient.Interface
	log    *logrus.Entry
}

//NewReconciler creates a new Reconciler
func NewReconciler(log *logrus.Entry, arocli aroclient.Interface) *AzureNSGReconciler {
	return &AzureNSGReconciler{
		arocli: arocli,
		log:    log,
	}
}

//Reconcile fixes the Network Security Groups
func (r *AzureNSGReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {

	return reconcile.Result{}, nil
}

//SetupWithManager creates the controller
func (r *AzureNSGReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		Owns(&appsv1.DaemonSet{}).
		Named(controllers.AzureNSGControllerName).
		Complete(r)
}
