package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	securityv1 "github.com/openshift/api/security/v1"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

// GenevaloggingReconciler reconciles a Cluster object
type GenevaloggingReconciler struct {
	kubernetescli kubernetes.Interface
	securitycli   securityclient.Interface
	arocli        aroclient.Interface
	restConfig    *rest.Config
	log           *logrus.Entry
}

func NewReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface, securitycli securityclient.Interface, arocli aroclient.Interface, restConfig *rest.Config) *GenevaloggingReconciler {
	return &GenevaloggingReconciler{
		securitycli:   securitycli,
		kubernetescli: kubernetescli,
		arocli:        arocli,
		restConfig:    restConfig,
		log:           log,
	}
}

// Reconcile the genevalogging deployment.
func (r *GenevaloggingReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	// TODO(mj): controller-runtime master fixes the need for this (https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/reconcile/reconcile.go#L93) but it's not yet released.
	ctx := context.Background()

	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	mysec, err := r.kubernetescli.CoreV1().Secrets(operator.Namespace).Get(ctx, operator.SecretName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	// TODO: dh should be a field in r, but the fact that it is initialised here
	// each time currently saves us in the case that the controller runs before
	// the SCC API is registered.
	dh, err := dynamichelper.New(r.log, r.restConfig)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	resources, err := r.resources(ctx, instance, mysec.Data[GenevaCertName], mysec.Data[GenevaKeyName])
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

	err = dh.Ensure(ctx, resources...)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager setup our manager
func (r *GenevaloggingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(meta metav1.Object, object runtime.Object) bool {
		return meta.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Namespace{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&securityv1.SecurityContextConstraints{}).
		Named(controllers.GenevaLoggingControllerName).
		Complete(r)
}
