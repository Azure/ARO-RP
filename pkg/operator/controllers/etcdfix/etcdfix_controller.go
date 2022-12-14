package etcdfix

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	securityv1 "github.com/openshift/api/security/v1"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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
	ControllerName = "etcdfixController"
)

type Reconciler struct {
	log *logrus.Entry

	arocli        aroclient.Interface
	dynamicHelper dynamichelper.Interface
	restConfig    *rest.Config
	securitycli   securityclient.Interface
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, restConfig *rest.Config, securitycli securityclient.Interface) *Reconciler {
	return &Reconciler{
		log:         log,
		arocli:      arocli,
		restConfig:  restConfig,
		securitycli: securitycli,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	r.deploy(ctx, instance)

	r.log.Debug("running")
	return ctrl.Result{}, nil
}

func (r *Reconciler) deploy(ctx context.Context, instance *arov1alpha1.Cluster) (ctrl.Result, error) {
	var err error
	r.dynamicHelper, err = dynamichelper.New(r.log, r.restConfig)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	resources, err := r.resources(ctx, instance)
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

	err = r.dynamicHelper.Ensure(ctx, resources...)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager creates the controller
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Owns(&corev1.Namespace{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&securityv1.SecurityContextConstraints{}).
		Named(ControllerName).
		Complete(r)
}
