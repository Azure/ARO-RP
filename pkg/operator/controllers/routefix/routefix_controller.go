package routefix

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	securityv1 "github.com/openshift/api/security/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	CONFIG_NAMESPACE string = "aro.routefix"
	ENABLED          string = CONFIG_NAMESPACE + ".enabled"
	MANAGED          string = CONFIG_NAMESPACE + ".managed"
)

// Reconciler is the controller struct
type Reconciler struct {
	log *logrus.Entry

	arocli        aroclient.Interface
	configcli     configclient.Interface
	kubernetescli kubernetes.Interface
	securitycli   securityclient.Interface

	restConfig *rest.Config
	verFixed46 *version.Version
	verFixed47 *version.Version
}

var (
	verFixed47, _ = version.ParseVersion("4.7.18")
	verFixed46, _ = version.ParseVersion("4.6.37")
)

// NewReconciler creates a new Reconciler
func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, configcli configclient.Interface, kubernetescli kubernetes.Interface, securitycli securityclient.Interface, restConfig *rest.Config) *Reconciler {
	return &Reconciler{
		log:           log,
		arocli:        arocli,
		configcli:     configcli,
		kubernetescli: kubernetescli,
		securitycli:   securitycli,
		restConfig:    restConfig,
		verFixed46:    verFixed46,
		verFixed47:    verFixed47,
	}
}

// Reconcile fixes the daemonset Routefix
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(ENABLED) {
		// controller is disabled
		return reconcile.Result{}, nil
	}

	// cluster version is not set to final until upgrade is completed. We need to
	// detect if desired version is with the fix, so we can prevent stuck upgrade
	// by deleting fix resources
	clusterVersion, err := version.GetClusterDesiredVersion(ctx, r.configcli)
	if err != nil {
		r.log.Errorf("error getting the OpenShift desired version: %v", err)
		return reconcile.Result{}, err
	}

	if r.isRequired(clusterVersion) {
		return r.deploy(ctx, instance)
	}
	return r.remove(ctx, instance)
}

func (r *Reconciler) deploy(ctx context.Context, instance *arov1alpha1.Cluster) (ctrl.Result, error) {
	// TODO: dh should be a field in r, but the fact that it is initialised here
	// each time currently saves us in the case that the controller runs before
	// the SCC API is registered.
	dh, err := dynamichelper.New(r.log, r.restConfig)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	resources, err := r.resources(ctx, instance)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	err = dynamichelper.SetControllerReferences(resources, instance)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	err = dynamichelper.Prepare(resources)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	err = dh.Ensure(ctx, resources...)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) remove(ctx context.Context, instance *arov1alpha1.Cluster) (ctrl.Result, error) {
	err := r.kubernetescli.CoreV1().Namespaces().Delete(ctx, kubeNamespace, metav1.DeleteOptions{})
	if !kerrors.IsNotFound(err) {
		return reconcile.Result{}, err
	}
	err = r.kubernetescli.RbacV1().ClusterRoleBindings().Delete(ctx, kubeName, metav1.DeleteOptions{})
	if kerrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, err
}

// SetupWithManager creates the controller
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Owns(&corev1.Namespace{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&securityv1.SecurityContextConstraints{}).
		Named(controllers.RouteFixControllerName).
		Complete(r)
}

func (r *Reconciler) isRequired(clusterVersion *version.Version) bool {
	y := clusterVersion.V[1]
	switch y {
	case 6: // 4.6.X
		return clusterVersion.Lt(r.verFixed46)
	case 7: // 4.7.X
		return clusterVersion.Lt(r.verFixed47)
	default:
		return clusterVersion.Lt(r.verFixed47)
	}
}
