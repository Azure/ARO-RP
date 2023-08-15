package ingresscertificatechecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	checkercommon "github.com/Azure/ARO-RP/pkg/operator/controllers/checkers/common"
	"github.com/Azure/ARO-RP/pkg/util/conditions"
)

// This is the permissions that this controller needs to work.
// "make generate" will run kubebuilder and cause operator/deploy/staticresources/*/role.yaml to be updated
// from the annotation below.
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

const (
	ControllerName = "IngressCertificateChecker"
)

// Reconciler runs a number of checkers
type Reconciler struct {
	log     *logrus.Entry
	role    string
	client  client.Client
	checker ingressCertificateChecker
}

func NewReconciler(log *logrus.Entry, client client.Client, role string) *Reconciler {
	return &Reconciler{
		log:     log,
		role:    role,
		client:  client,
		checker: newIngressCertificateChecker(client),
	}
}

// Reconcile will keep checking that the cluster has a valid default IngressController configuration.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	cluster := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !cluster.Spec.OperatorFlags.GetSimpleBoolean(checkercommon.ControllerEnabled) {
		r.log.Debug("controller is disabled")
		return r.reconcileDisabled(ctx)
	}

	r.log.Debug("running")

	if err != nil {
		return reconcile.Result{}, err
	}

	checkErr := r.checker.Check(ctx)
	condition := r.condition(checkErr)

	err = conditions.SetCondition(ctx, r.client, condition, r.role)
	if err != nil {
		return reconcile.Result{}, err
	}

	// In case of this error we want to set condition to False, but
	// we don't want to continuously try to reconcile it as it might
	// be expected config in some cases (e.g. custom domain cluster)
	if errors.Is(checkErr, errNoCertificateAndCustomDomain) {
		return reconcile.Result{RequeueAfter: time.Hour}, nil
	}

	// We always requeue here:
	// * Either immediately (with rate limiting) based on the error
	//   when checkErr != nil.
	// * Or based on RequeueAfter when err == nil.
	return reconcile.Result{RequeueAfter: time.Hour}, checkErr
}

func (r *Reconciler) reconcileDisabled(ctx context.Context) (ctrl.Result, error) {
	condition := &operatorv1.OperatorCondition{
		Type:   arov1alpha1.DefaultIngressCertificate,
		Status: operatorv1.ConditionUnknown,
	}

	return reconcile.Result{}, conditions.SetCondition(ctx, r.client, condition, r.role)
}

func (r *Reconciler) condition(checkErr error) *operatorv1.OperatorCondition {
	if checkErr != nil {
		return &operatorv1.OperatorCondition{
			Type:    arov1alpha1.DefaultIngressCertificate,
			Status:  operatorv1.ConditionFalse,
			Message: checkErr.Error(),
			Reason:  "CheckFailed",
		}
	}

	return &operatorv1.OperatorCondition{
		Type:    arov1alpha1.DefaultIngressCertificate,
		Status:  operatorv1.ConditionTrue,
		Message: "Default ingress certificate is in use",
		Reason:  "CheckDone",
	}
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	defaultIngressControllerPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetNamespace() == "openshift-ingress-operator" && o.GetName() == "default"
	})

	clusterVersionPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == "version"
	})

	builder := ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Watches(
			&source.Kind{Type: &operatorv1.IngressController{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(defaultIngressControllerPredicate),
		).
		Watches(
			&source.Kind{Type: &configv1.ClusterVersion{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(clusterVersionPredicate),
		)

	return builder.Named(ControllerName).Complete(r)
}
