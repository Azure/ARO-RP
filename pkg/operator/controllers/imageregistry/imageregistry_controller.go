package imageregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	imageregistryv1 "github.com/openshift/api/imageregistry/v1"
	imageregistryclient "github.com/openshift/client-go/imageregistry/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
)

const (
	ControllerName = "ImageRegistry"

	controllerEnabled = "aro.imageregistryconfig.enabled"
)

type Reconciler struct {
	log *logrus.Entry

	arocli           aroclient.Interface
	imageregistrycli imageregistryclient.Interface
	//kubernetescli kubernetes.Interface
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, imageregistryclient imageregistryclient.Interface) *Reconciler {
	return &Reconciler{
		log:              log,
		arocli:           arocli,
		imageregistrycli: imageregistryclient,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	// Validate operator is enabled via cluster feature flag
	cluster, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}
	if request.Name != "cluster" || !cluster.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		return reconcile.Result{}, nil
	}

	// Correct the DisableRedirect setting
	registryConfig, err := r.imageregistrycli.ImageregistryV1().Configs().Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}
	if !registryConfig.Spec.DisableRedirect {
		r.log.Info("An attempt was made to enable redirect! Disabling...")
		registryConfig.Spec.DisableRedirect = true
		r.imageregistrycli.ImageregistryV1().Configs().Update(ctx, registryConfig, metav1.UpdateOptions{})
		return reconcile.Result{}, nil
	}

	// Otherwise, do nothing
	return reconcile.Result{}, nil
}

// SetupWithManager setup our mananger
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&imageregistryv1.Config{}).
		Named(ControllerName).
		Complete(r)
}
