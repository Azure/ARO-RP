package configs

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

type Reconciler struct {
	client.Client
	Scheme *kruntime.Scheme

	log     *logrus.Entry
	arocli  aroclient.Interface
	configs []Config
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, mgr ctrl.Manager) *Reconciler {

	configs := []Config{
		NewAutoNodeSizeConfig(),
	}

	return &Reconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),

		log:     log,
		configs: configs,
		arocli:  arocli,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	var aro arov1alpha1.Cluster
	var err error

	err = r.Get(ctx, request.NamespacedName, &aro)
	if err != nil {
		r.log.Error("unable to fetch aro cluster")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.log.Infof("Config changed, autoSize: %t\n", aro.Spec.Features.ReconcileAutoSizedNodes)

	for _, conf := range r.configs {
		if conf.IsApplicable(aro, r, ctx) {
			err = conf.Ensure(r, ctx)
		} else {
			err = conf.Remove(r, ctx)
		}

		if err != nil {
			r.log.Errorf("config %s returned error: %v", conf.Name(), err)
			return reconcile.Result{}, client.IgnoreNotFound(err)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager prepares the controller with info who to watch
// TODO: Add informers with a cache to reduce the amount of API calls
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {

	builder := ctrl.NewControllerManagedBy(mgr).For(&arov1alpha1.Cluster{})

	for _, config := range r.configs {
		builder = config.AddOwns(builder)
	}

	return builder.
		Named(controllers.ConfigsControllerName).
		Complete(r)
}
