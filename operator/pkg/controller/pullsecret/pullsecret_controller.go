package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Azure/ARO-RP/operator/pkg/controller/consts"
)

var (
	log            = logf.Log.WithName("controller_pullsecret")
	pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}
)

// Add creates a new Secret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &secretReconciler{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("pull-secret-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to secrets
	return c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{})
}

// blank assignment to verify that secretReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &secretReconciler{}

// secretReconciler reconciles the Pull Secret object
type secretReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the pull-secret object and makes sure the ACR
// repository is always configured
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *secretReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if request.NamespacedName != pullSecretName {
		// filter out other secrets.
		return consts.ReconcileResultIgnore, nil
	}

	log.Info("Reconciling pull-secret")

	ctx := context.TODO()
	isCreate := false
	ps := &corev1.Secret{}
	err := r.client.Get(ctx, request.NamespacedName, ps)
	if err != nil && errors.IsNotFound(err) {
		isCreate = true
		ps = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pullSecretName.Name,
				Namespace: pullSecretName.Namespace,
			},
			Type: v1.SecretTypeDockerConfigJson,
		}
	} else if err != nil {
		log.Error(err, "failed to Get pull secret")
		return consts.ReconcileResultError, err
	}

	changed, err := r.pullSecretRepair(ps)
	if err != nil {
		return consts.ReconcileResultError, err
	}
	if !isCreate && !changed {
		log.Info("Skip reconcile: Pull Secret repair not required")
		return consts.ReconcileResultDone, nil
	}
	if isCreate {
		log.Info("Re-creating the Pull Secret")
		err = r.client.Create(ctx, ps)
	} else if changed {
		log.Info("Updating the Pull Secret")
		err = r.client.Update(ctx, ps)
	}
	if err != nil {
		log.Error(err, "Failed to repair the Pull Secret")
		return consts.ReconcileResultError, err
	}
	log.Info("done, requeueing")
	return consts.ReconcileResultDone, nil
}

func (r *secretReconciler) pullSecretRepair(cr *corev1.Secret) (bool, error) {
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

	newPS, changed, err := repair(cr.Data[corev1.DockerConfigJsonKey], psPath)
	if err != nil {
		return false, err
	}
	if changed {
		cr.Data[corev1.DockerConfigJsonKey] = newPS
	}
	return changed, nil
}
