package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aroclient "github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

// PullsecretReconciler reconciles a Cluster object
type PullsecretReconciler struct {
	Kubernetescli           kubernetes.Interface
	AROCli                  aroclient.AroV1alpha1Interface
	Log                     *logrus.Entry
	Scheme                  *runtime.Scheme
	requiredRepoTokensStore map[string]string
}

// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update;patch;create

func (r *PullsecretReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	if request.NamespacedName != pullSecretName {
		// filter out other secrets.
		return reconcile.Result{}, nil
	}
	if len(r.requiredRepoTokensStore) == 0 {
		// nothing to do.
		return reconcile.Result{}, nil
	}
	r.Log.Info("Reconciling pull-secret")

	return reconcile.Result{}, retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ps, isCreate, err := r.pullsecret(request)
		if err != nil {
			return err
		}

		// validate
		if !json.Valid(ps.Data[v1.DockerConfigJsonKey]) {
			delete(ps.Data, v1.DockerConfigJsonKey)
		}
		if ps.Data == nil {
			ps.Data = map[string][]byte{}
		}

		// repair data
		newPS, changed, err := pullsecret.Replace(ps.Data[corev1.DockerConfigJsonKey], r.requiredRepoTokensStore)
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
			changed = true

			// unfortunately the type field is immutable.
			err = r.Kubernetescli.CoreV1().Secrets(ps.Namespace).Delete(ps.Name, nil)
			if err != nil {
				return err
			}

			// there is a small risk of crashing here: if that happens, we will
			// restart, create a new pull secret, and will have dropped the rest
			// of the customer's pull secret on the floor :-(
		}
		if !changed {
			r.Log.Info("Skip reconcile: Pull Secret repair not required")
			return nil
		}

		ps.Data[corev1.DockerConfigJsonKey] = newPS

		if isCreate {
			r.Log.Info("Re-creating the Pull Secret")
			_, err = r.Kubernetescli.CoreV1().Secrets("openshift-config").Create(ps)
		} else {
			r.Log.Info("Updating the Pull Secret")
			_, err = r.Kubernetescli.CoreV1().Secrets("openshift-config").Update(ps)
		}
		return err
	})
}

func (r *PullsecretReconciler) pullsecret(request ctrl.Request) (*v1.Secret, bool, error) {
	var isCreate bool
	ps, err := r.Kubernetescli.CoreV1().Secrets(request.Namespace).Get(request.Name, metav1.GetOptions{})
	switch {
	case apierrors.IsNotFound(err):
		ps = &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      request.Name,
				Namespace: request.Namespace,
			},
			Type: v1.SecretTypeDockerConfigJson,
		}
		isCreate = true
	case err != nil:
		return nil, false, err
	}
	return ps, isCreate, nil
}

func (r *PullsecretReconciler) requiredRepoTokens() (map[string]string, error) {
	// The idea here is you mount a secret as a file under /pull-secrets with
	// the same name as the registry in the pull secret.
	psPath := "/pull-secrets"
	if os.Getenv("RP_MODE") == "development" {
		pathOverride := os.Getenv("PULL_SECRET_PATH") // for development
		if pathOverride != "" {
			psPath = pathOverride
			r.Log.Warnf("running outside the cluster, using override path %s", pathOverride)
		} else {
			r.Log.Warnf("running outside the cluster, disabling pull secret controller")
			return map[string]string{}, nil
		}
	}
	repoTokens := map[string]string{}

	files, err := ioutil.ReadDir(psPath)
	if err != nil {
		return nil, err
	}
	for _, fName := range files {
		fpath := path.Join(psPath, fName.Name())
		if fName.IsDir() || strings.HasPrefix(fName.Name(), "..") {
			continue
		}
		data, err := ioutil.ReadFile(fpath)
		if err != nil {
			return nil, err
		}
		if len(data) > 0 {
			r.Log.Infof("requiredRepo: %s", fpath)
			repoTokens[fName.Name()] = base64.StdEncoding.EncodeToString(data)
		}
	}
	return repoTokens, nil
}

func triggerReconcile(secret *corev1.Secret) bool {
	return secret.Name == pullSecretName.Name && secret.Namespace == pullSecretName.Namespace
}

func (r *PullsecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	r.requiredRepoTokensStore, err = r.requiredRepoTokens()
	if err != nil {
		return err
	}
	isPullSecret := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSecret, ok := e.ObjectOld.(*corev1.Secret)
			if !ok {
				return false
			}
			newSecret, ok := e.ObjectNew.(*corev1.Secret)
			if !ok {
				return false
			}
			return (triggerReconcile(oldSecret) || triggerReconcile(newSecret))
		},
		CreateFunc: func(e event.CreateEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			if !ok {
				return false
			}
			return triggerReconcile(secret)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			if !ok {
				return false
			}
			return triggerReconcile(secret)
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Secret{}).WithEventFilter(isPullSecret).
		Complete(r)
}
