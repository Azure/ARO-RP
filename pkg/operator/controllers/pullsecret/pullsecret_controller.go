package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

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
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

// PullSecretReconciler reconciles a Cluster object
type PullSecretReconciler struct {
	kubernetescli kubernetes.Interface
	arocli        aroclient.AroV1alpha1Interface
	log           *logrus.Entry
}

func NewReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface, arocli aroclient.AroV1alpha1Interface) *PullSecretReconciler {
	return &PullSecretReconciler{
		log:           log,
		kubernetescli: kubernetescli,
		arocli:        arocli,
	}
}

// Reconcile will make sure that the ACR part of the pull secret is correct. The
// conditions under which Reconcile is called are slightly unusual and are as
// follows:
// * If the Cluster object changes, we'll see the *Cluster* object requested.
// * If a Secret object owned by the Cluster object changes (e.g., but not
//   limited to, the configuration Secret, we'll see the *Cluster* object
//   requested).
// * If the pull Secret object (which is not owned by the Cluster object)
//   changes, we'll see the pull Secret object requested.
func (r *PullSecretReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	// TODO(mj): controller-runtime master fixes the need for this (https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/reconcile/reconcile.go#L93) but it's not yet released.
	ctx := context.Background()
	if request.NamespacedName != pullSecretName &&
		request.Name != arov1alpha1.SingletonClusterName {
		return reconcile.Result{}, nil
	}

	mysec, err := r.kubernetescli.CoreV1().Secrets(operator.Namespace).Get(ctx, operator.SecretName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ps, isCreate, err := r.pullsecret(ctx)
		if err != nil {
			return err
		}

		if ps.Data == nil {
			ps.Data = map[string][]byte{}
		}

		// validate
		if !json.Valid(ps.Data[v1.DockerConfigJsonKey]) {
			r.log.Info("pull secret is not valid json - recreating")
			delete(ps.Data, v1.DockerConfigJsonKey)
		}

		pullsec, changed, err := pullsecret.Merge(string(ps.Data[corev1.DockerConfigJsonKey]), string(mysec.Data[v1.DockerConfigJsonKey]))
		if err != nil {
			return err
		}

		// repair Secret type
		if ps.Type != v1.SecretTypeDockerConfigJson {
			ps = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pullSecretName.Name,
					Namespace: pullSecretName.Namespace,
				},
				Type: v1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{},
			}
			isCreate = true
			r.log.Info("pull secret has the wrong type - recreating")

			// unfortunately the type field is immutable.
			err = r.kubernetescli.CoreV1().Secrets(ps.Namespace).Delete(ctx, ps.Name, metav1.DeleteOptions{})
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
			r.log.Info("re-creating pull secret")
			_, err = r.kubernetescli.CoreV1().Secrets(ps.Namespace).Create(ctx, ps, metav1.CreateOptions{})
		} else {
			r.log.Info("updating pull secret")
			_, err = r.kubernetescli.CoreV1().Secrets(ps.Namespace).Update(ctx, ps, metav1.UpdateOptions{})
		}
		return err
	})
}

func (r *PullSecretReconciler) pullsecret(ctx context.Context) (*v1.Secret, bool, error) {
	ps, err := r.kubernetescli.CoreV1().Secrets(pullSecretName.Namespace).Get(ctx, pullSecretName.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pullSecretName.Name,
				Namespace: pullSecretName.Namespace,
			},
			Type: v1.SecretTypeDockerConfigJson,
		}, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return ps, false, nil
}

func triggerReconcile(secret *corev1.Secret) bool {
	return (secret.Name == pullSecretName.Name && secret.Namespace == pullSecretName.Namespace) ||
		(secret.Name == operator.SecretName && secret.Namespace == operator.Namespace)
}

// SetupWithManager setup our mananger
func (r *PullSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	isPullSecret := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			_, ok := e.ObjectOld.(*arov1alpha1.Cluster)
			if ok {
				return true
			}

			secret, ok := e.ObjectOld.(*corev1.Secret)
			return ok && triggerReconcile(secret)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			_, ok := e.Object.(*arov1alpha1.Cluster)
			if ok {
				return true
			}

			secret, ok := e.Object.(*corev1.Secret)
			return ok && triggerReconcile(secret)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			_, ok := e.Object.(*arov1alpha1.Cluster)
			if ok {
				return true
			}

			secret, ok := e.Object.(*corev1.Secret)
			return ok && triggerReconcile(secret)
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		// https://github.com/kubernetes-sigs/controller-runtime/issues/1173
		// equivalent to For(&v1.Secret{})., but can't call For multiple times on one builder
		Watches(&source.Kind{Type: &v1.Secret{}}, &handler.EnqueueRequestForObject{}).
		Owns(&v1.Secret{}).
		WithEventFilter(isPullSecret).
		Named(controllers.PullSecretControllerName).
		Complete(r)
}
