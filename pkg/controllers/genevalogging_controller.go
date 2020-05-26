package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/genevalogging"
	aroclient "github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset/versioned/typed/aro.openshift.io/v1alpha1"
)

// GenevaloggingReconciler reconciles a Cluster object
type GenevaloggingReconciler struct {
	Kubernetescli kubernetes.Interface
	Securitycli   securityclient.Interface
	AROCli        aroclient.AroV1alpha1Interface
	RestConfig    *rest.Config
	Log           *logrus.Entry
	Scheme        *runtime.Scheme
}

// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;create;update
// +kubebuilder:rbac:groups="",resources=namespaces;serviceaccounts;configmaps,verbs=get;create;update
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;create;update

func (r *GenevaloggingReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	if request.Name != aro.SingletonClusterName {
		return reconcile.Result{}, nil
	}
	r.Log.Info("Reconciling genevalogging deployment")

	ctx := context.TODO()
	instance, err := r.AROCli.Clusters().Get(request.Name, metav1.GetOptions{})
	if err != nil {
		// Error reading the object or not found - requeue the request.
		return reconcile.Result{}, err
	}

	newCert, err := r.certificatesSecret(instance)
	if err != nil {
		r.Log.Error(err)
		return reconcile.Result{}, err
	}
	dh, err := dynamichelper.New(r.Log, r.RestConfig, dynamichelper.UpdatePolicy{
		IgnoreDefaults:  true,
		LogChanges:      true,
		RetryOnConflict: true,
	})
	if err != nil {
		r.Log.Error(err)
		return reconcile.Result{}, err
	}
	gl := genevalogging.New(r.Log, &instance.Spec, dh, r.Securitycli, newCert)
	err = gl.CreateOrUpdate(ctx)
	if err != nil {
		r.Log.Error(err)
		return reconcile.Result{}, err
	}

	r.Log.Info("done, requeueing")
	return ReconcileResultRequeue, nil
}

func (r *GenevaloggingReconciler) certificatesSecret(instance *aro.Cluster) (*v1.Secret, error) {
	newCert, err := r.Kubernetescli.CoreV1().Secrets(instance.Spec.GenevaLogging.Namespace).Get("certificates", metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		// copy the certificates from our namespace into the genevalogging one.
		certs, err := r.Kubernetescli.CoreV1().Secrets(OperatorNamespace).Get("certificates", metav1.GetOptions{})
		if err != nil {
			r.Log.Errorf("Error reading the certificates secret: %v", err)
			return nil, err
		}
		newCert = certs.DeepCopy()
		newCert.Namespace = instance.Spec.GenevaLogging.Namespace
	}
	return newCert, nil
}

func (r *GenevaloggingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aro.Cluster{}).
		Complete(r)
	// TODO can we watch the genevalogging resources?
}
