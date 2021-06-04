package image

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

const imageConfigResource = "cluster"

var allowedRegistries = []string{"arosvc.azurecr.io", "quay.io"}

type Reconciler struct {
	arocli        aroclient.Interface
	kubernetescli kubernetes.Interface
	configcli     configclient.Interface
	log           *logrus.Entry
	jsonHandle    *codec.JsonHandle
}

func NewReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface, arocli aroclient.Interface) *Reconciler {
	return &Reconciler{
		arocli:        arocli,
		kubernetescli: kubernetescli,
		log:           log,
		jsonHandle:    new(codec.JsonHandle),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	imageconfig, err := r.configcli.ConfigV1().Images().Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	var regMap = make(map[string]bool)

	for _, registry := range allowedRegistries {
		regMap[registry] = false
	}

	for _, allowedRegistry := range imageconfig.Spec.RegistrySources.AllowedRegistries {
		if _, ok := regMap[allowedRegistry]; ok {
			regMap[allowedRegistry] = true
		}
	}

	return reconcile.Result{}, nil
}

// SetupWithManager setup the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.log.Info("starting image controller")

	imagePredicate := predicate.NewPredicateFuncs(func(meta metav1.Object, object runtime.Object) bool {
		return meta.GetName() == imageConfigResource
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&configv1.Image{}, builder.WithPredicates(imagePredicate)).
		Named(controllers.MonitoringControllerName).
		Complete(r)
}
