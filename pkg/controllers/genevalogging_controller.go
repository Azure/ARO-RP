package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

// This is the permissions that this controller needs to work.
// "make generate" will run kubebuilder and cause operator/deploy/staticresources/role.yaml to be updated
// from the annotation below.
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters;clusters/finalizers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets;daemonsets,verbs=list;watch;get;create;update
// +kubebuilder:rbac:groups="",resources=namespaces;namespaces;serviceaccounts;serviceaccounts;configmaps;configmaps,verbs=get;create;update
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints;securitycontextconstraints,verbs=get;create;update

// Reconcile the genevalogging deployment.
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
	gl := genevalogging.New(r.Log, &instance.Spec, r.Securitycli, newCert)

	resources, err := gl.Resources(ctx)
	if err != nil {
		r.Log.Error(err)
		return reconcile.Result{}, err
	}
	for _, res := range resources {
		un, err := dh.ToUnstructured(res)
		if err != nil {
			r.Log.Error(err)
			return reconcile.Result{}, err
		}

		if un.GetKind() != "Namespace" {
			// This sets the reference on all objects that we create
			// to our cluster instance. This causes the Owns() below to work and
			// to get Reconcile events when anything happens to our objects.
			err = controllerutil.SetControllerReference(instance, un, r.Scheme)
			if err != nil {
				r.Log.Errorf("SetControllerReference %s/%s: %v", instance.Kind, instance.Name, err)
				return reconcile.Result{}, err
			}
		}

		err = dh.CreateOrUpdate(ctx, un)
		if err != nil {
			r.Log.Error(err)
			return reconcile.Result{}, err
		}
	}

	r.Log.Info("done, requeueing")
	// watching should catch all changes, but double check later..
	return ReconcileResultRequeueLong, nil
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

		newCert = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "certificates",
				Namespace: instance.Spec.GenevaLogging.Namespace,
			},
		}
		for k, v := range certs.StringData {
			newCert.StringData[k] = v
		}
	} else if err != nil {
		return nil, err
	}
	return newCert, nil
}

// SetupWithManager setup our mananger
func (r *GenevaloggingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aro.Cluster{}).Owns(&appsv1.DaemonSet{}).Owns(&corev1.Secret{}).
		Complete(r)
}
