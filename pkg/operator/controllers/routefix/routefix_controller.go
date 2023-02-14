package routefix

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	securityv1 "github.com/openshift/api/security/v1"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	ControllerName = "RouteFix"

	controllerEnabled = "aro.routefix.enabled"
)

// Reconciler is the controller struct
type Reconciler struct {
	log *logrus.Entry

	client client.Client
	dh     dynamichelper.Interface

	verFixed46 *version.Version
	verFixed47 *version.Version
}

var (
	verFixed47, _ = version.ParseVersion("4.7.18")
	verFixed46, _ = version.ParseVersion("4.6.37")
)

// NewReconciler creates a new Reconciler
func NewReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *Reconciler {
	return &Reconciler{
		log: log,

		client: client,
		dh:     dh,

		verFixed46: verFixed46,
		verFixed47: verFixed47,
	}
}

// Reconcile fixes the daemonset Routefix
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")

	// cluster version is not set to final until upgrade is completed. We need to
	// detect if desired version is with the fix, so we can prevent stuck upgrade
	// by deleting fix resources
	cv := &configv1.ClusterVersion{}
	err = r.client.Get(ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterVersion, err := version.ParseVersion(cv.Status.Desired.Version)
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
	r.log.Debugf("deploying RouteFix")

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

	err = r.dh.Ensure(ctx, resources...)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) remove(ctx context.Context, instance *arov1alpha1.Cluster) (ctrl.Result, error) {
	r.log.Debugf("removing RouteFix")

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubeNamespace,
		},
	}
	err := r.client.Delete(ctx, ns)
	if !kerrors.IsNotFound(err) {
		return reconcile.Result{}, err
	}

	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubeName,
		},
	}
	err = r.client.Delete(ctx, crb)
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
		Named(ControllerName).
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
