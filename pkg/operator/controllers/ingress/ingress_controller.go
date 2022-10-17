package ingress

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/to"
	operatorv1 "github.com/openshift/api/operator/v1"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
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
)

const (
	ControllerName = "Ingress"

	controllerEnabled                   = "aro.ingress.enabled"
	openshiftIngressControllerNamespace = "openshift-ingress-operator"
	openshiftIngressControllerName      = "default"
	minimumReplicas                     = 2
)

// Reconciler spots openshift ingress controllers has abnormal replica counts (less than 2)
// when happens, it tries to rescale the controller to 2 replicas, i.e., the minimum required replicas
type Reconciler struct {
	log *logrus.Entry

	arocli      aroclient.Interface
	operatorcli operatorclient.Interface
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, operatorcli operatorclient.Interface) *Reconciler {
	return &Reconciler{
		log:         log,
		arocli:      arocli,
		operatorcli: operatorcli,
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

	ingress, err := r.operatorcli.OperatorV1().IngressControllers(openshiftIngressControllerNamespace).Get(ctx, openshiftIngressControllerName, metav1.GetOptions{})
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	if ingress.Spec.Replicas != nil && *ingress.Spec.Replicas < minimumReplicas {
		ingress.Spec.Replicas = to.Int32Ptr(minimumReplicas)
		_, err := r.operatorcli.OperatorV1().IngressControllers(openshiftIngressControllerNamespace).Update(ctx, ingress, metav1.UpdateOptions{})
		if err != nil {
			r.log.Error(err)
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// SetupWithManager setup the mananger for openshift ingress controller resource
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Watches(&source.Kind{Type: &operatorv1.IngressController{}}, &handler.EnqueueRequestForObject{}).
		Named(ControllerName).
		Complete(r)
}
