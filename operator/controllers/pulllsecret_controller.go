package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arov1alpha1 "github.com/Azure/ARO-RP/operator/api/v1alpha1"
	"github.com/Azure/ARO-RP/operator/controllers/pullsecret"
)

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

// PullsecretReconciler reconciles a Cluster object
type PullsecretReconciler struct {
	client.Client
	Log    *logrus.Entry
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

func (r *PullsecretReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	if request.NamespacedName != pullSecretName {
		// filter out other secrets.
		return ReconcileResultIgnore, nil
	}

	r.Log.Info("Reconciling pull-secret")

	ctx := context.TODO()
	isCreate := false
	ps := &corev1.Secret{}
	err := r.Client.Get(ctx, request.NamespacedName, ps)
	if err != nil && errors.IsNotFound(err) {
		isCreate = true
		ps = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pullSecretName.Name,
				Namespace: pullSecretName.Namespace,
			},
			Type: corev1.SecretTypeDockerConfigJson,
		}
	} else if err != nil {
		r.Log.Error(err, "failed to Get pull secret")
		return ReconcileResultError, err
	}

	changed, err := r.pullSecretRepair(ps)
	if err != nil {
		return ReconcileResultError, err
	}
	if !isCreate && !changed {
		r.Log.Info("Skip reconcile: Pull Secret repair not required")
		return ReconcileResultDone, nil
	}
	if isCreate {
		r.Log.Info("Re-creating the Pull Secret")
		err = r.Client.Create(ctx, ps)
	} else if changed {
		r.Log.Info("Updating the Pull Secret")
		err = r.Client.Update(ctx, ps)
	}
	if err != nil {
		r.Log.Error(err, "Failed to repair the Pull Secret")
		return ReconcileResultError, err
	}
	r.Log.Info("done, requeueing")
	return ReconcileResultDone, nil
}

func (r *PullsecretReconciler) pullSecretRepair(cr *corev1.Secret) (bool, error) {
	if cr.Data == nil {
		cr.Data = map[string][]byte{}
	}

	// The idea here is you mount a secret as a file under /pull-secrets with
	// the same name as the registry in the pull secret.
	psPath := "/pull-secrets"
	pathOverride := os.Getenv("PULL_SECRET_PATH") // for development
	if pathOverride != "" {
		psPath = pathOverride
	}

	newPS, changed, err := pullsecret.Repair(cr.Data[corev1.DockerConfigJsonKey], psPath)
	if err != nil {
		return false, err
	}
	if changed {
		cr.Data[corev1.DockerConfigJsonKey] = newPS
	}
	return changed, nil
}

func (r *PullsecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		Complete(r)
}
