package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	securityv1 "github.com/openshift/api/security/v1"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	ControllerName = "GenevaLogging"

	controllerEnabled = "aro.genevalogging.enabled"
	// full pullspec of fluentbit image
	controllerFluentbitPullSpec = "aro.genevalogging.fluentbit.pullSpec"
	// full pullspec of mdsd image
	controllerMDSDPullSpec = "aro.genevalogging.mdsd.pullSpec"
)

// Reconciler reconciles a Cluster object
type Reconciler struct {
	log *logrus.Entry

	dh     dynamichelper.Interface
	client client.Client
}

func NewReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *Reconciler {
	return &Reconciler{
		log: log,

		dh:     dh,
		client: client,
	}
}

// Reconcile the genevalogging deployment.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")
	operatorSecret := &corev1.Secret{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: operator.Namespace, Name: operator.SecretName}, operatorSecret)
	if err != nil {
		return reconcile.Result{}, err
	}

	resources, err := r.resources(ctx, instance, operatorSecret.Data[GenevaCertName], operatorSecret.Data[GenevaKeyName])
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	err = dynamichelper.SetControllerReferences(resources, instance)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	err = dynamichelper.Prepare(resources)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	err = r.dh.Ensure(ctx, resources...)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Namespace{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&securityv1.SecurityContextConstraints{}).
		Named(ControllerName).
		Complete(r)
}
