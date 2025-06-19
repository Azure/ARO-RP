package ingress

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
)

const (
	ControllerName                      = "IngressControllerARO"
	openshiftIngressControllerNamespace = "openshift-ingress-operator"
	openshiftIngressControllerName      = "default"
	minimumReplicas                     = 2
)

// Reconciler spots openshift ingress controllers has abnormal replica counts (less than 2)
// when happens, it tries to rescale the controller to 2 replicas, i.e., the minimum required replicas
type Reconciler struct {
	base.AROController
}

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

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.IngressEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")
	ingress := &operatorv1.IngressController{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: openshiftIngressControllerNamespace, Name: openshiftIngressControllerName}, ingress)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	if ingress.Spec.Replicas != nil && *ingress.Spec.Replicas < minimumReplicas {
		ingress.Spec.Replicas = to.Int32Ptr(minimumReplicas)
		err := r.Client.Update(ctx, ingress)
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)
			return reconcile.Result{}, err
		}
	}

	r.ClearConditions(ctx)
	return reconcile.Result{}, nil
}

// SetupWithManager setup the mananger for openshift ingress controller resource
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Named(ControllerName).
		Complete(r)
}
