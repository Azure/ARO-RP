package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/controllers/pullsecret"
	aroclient "github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset/versioned/typed/aro.openshift.io/v1alpha1"
)

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

// PullsecretReconciler reconciles a Cluster object
type PullsecretReconciler struct {
	Kubernetescli kubernetes.Interface
	AROCli        aroclient.AroV1alpha1Interface
	Log           *logrus.Entry
	Scheme        *runtime.Scheme
}

// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

func (r *PullsecretReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	if request.NamespacedName != pullSecretName {
		// filter out other secrets.
		return ReconcileResultIgnore, nil
	}

	r.Log.Info("Reconciling pull-secret")

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var isCreate bool
		ps, err := r.Kubernetescli.CoreV1().Secrets(request.Namespace).Get(request.Name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			ps = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      request.Name,
					Namespace: request.Namespace,
				},
				Type: v1.SecretTypeDockerConfigJson,
			}
			isCreate = true
		case err != nil:
			return err
		}

		changed, err := r.pullSecretRepair(ps)
		if err != nil {
			return err
		}

		if !changed {
			r.Log.Info("Skip reconcile: Pull Secret repair not required")
			return nil
		}

		if isCreate {
			r.Log.Info("Re-creating the Pull Secret")
			_, err = r.Kubernetescli.CoreV1().Secrets("openshift-config").Create(ps)
		} else {
			r.Log.Info("Updating the Pull Secret")
			_, err = r.Kubernetescli.CoreV1().Secrets("openshift-config").Update(ps)
		}
		return err
	})
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
		For(&aro.Cluster{}).
		Complete(r)
}
