package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/deploy"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

// PullSecretReconciler reconciles a Cluster object
type PullSecretReconciler struct {
	kubernetescli kubernetes.Interface
	arocli        aroclient.AroV1alpha1Interface
	log           *logrus.Entry
}

func NewPullSecretReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface, arocli aroclient.AroV1alpha1Interface) *PullSecretReconciler {
	return &PullSecretReconciler{
		log:           log,
		kubernetescli: kubernetescli,
		arocli:        arocli,
	}
}

// Reconcile will make sure that the ACR part of the pull secret is correct
func (r *PullSecretReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	if request.NamespacedName != pullSecretName {
		return reconcile.Result{}, nil
	}

	pullsec, err := r.requiredPullSecret()
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ps, isCreate, err := r.pullsecret(request)
		if err != nil {
			return err
		}

		if ps.Data == nil {
			ps.Data = map[string][]byte{}
		}

		// validate
		if !json.Valid(ps.Data[v1.DockerConfigJsonKey]) {
			r.log.Info("Pull Secret is not valid json - recreating")
			delete(ps.Data, v1.DockerConfigJsonKey)
		}

		pullsec, changed, err := pullsecret.Merge(string(ps.Data[corev1.DockerConfigJsonKey]), pullsec)
		if err != nil {
			return err
		}

		// repair Secret type
		if ps.Type != v1.SecretTypeDockerConfigJson {
			ps = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pull-secret",
					Namespace: "openshift-config",
				},
				Type: v1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{},
			}
			isCreate = true
			r.log.Info("Pull Secret has the wrong secret type - recreating")

			// unfortunately the type field is immutable.
			err = r.kubernetescli.CoreV1().Secrets(ps.Namespace).Delete(ps.Name, nil)
			if err != nil {
				return err
			}

			// there is a small risk of crashing here: if that happens, we will
			// restart, create a new pull secret, and will have dropped the rest
			// of the customer's pull secret on the floor :-(
		}
		if !isCreate && !changed {
			return nil
		}

		ps.Data[corev1.DockerConfigJsonKey] = []byte(pullsec)

		if isCreate {
			r.log.Info("re-creating the Pull Secret")
			_, err = r.kubernetescli.CoreV1().Secrets("openshift-config").Create(ps)
		} else {
			r.log.Info("updating the Pull Secret")
			_, err = r.kubernetescli.CoreV1().Secrets("openshift-config").Update(ps)
		}
		return err
	})
}

func (r *PullSecretReconciler) pullsecret(request ctrl.Request) (*v1.Secret, bool, error) {
	ps, err := r.kubernetescli.CoreV1().Secrets(request.Namespace).Get(request.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      request.Name,
				Namespace: request.Namespace,
			},
			Type: v1.SecretTypeDockerConfigJson,
		}, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return ps, false, nil
}

func (r *PullSecretReconciler) requiredPullSecret() (string, error) {
	s, err := r.kubernetescli.CoreV1().Secrets(operator.Namespace).Get(deploy.ACRPullSecretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("Error reading the repoToken secret: %v", err)
	}

	return string(s.Data[v1.DockerConfigJsonKey]), nil
}

func triggerReconcile(secret *corev1.Secret) bool {
	return secret.Name == pullSecretName.Name && secret.Namespace == pullSecretName.Namespace
}

// SetupWithManager setup our mananger
func (r *PullSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// The pull secret may already be deleted when controller starts
	initialRequest := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: pullSecretName.Namespace,
			Name:      pullSecretName.Name,
		},
	}
	_, isCreate, err := r.pullsecret(initialRequest)
	if err == nil && isCreate {
		r.Reconcile(initialRequest)
	}

	isPullSecret := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			secret, ok := e.ObjectOld.(*corev1.Secret)
			return ok && triggerReconcile(secret)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			return ok && triggerReconcile(secret)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			return ok && triggerReconcile(secret)
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Secret{}).
		WithEventFilter(isPullSecret).
		Named(PullSecretControllerName).
		Complete(r)
}
