package clusterdnschecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/conditions"
)

// This is the permissions that this controller needs to work.
// "make generate" will run kubebuilder and cause operator/deploy/staticresources/*/role.yaml to be updated
// from the annotation below.
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

const (
	ControllerName = "ClusterDNSChecker"
)

// Reconciler runs a number of checkers
type Reconciler struct {
	log  *logrus.Entry
	role string

	checker clusterDNSChecker

	client client.Client
}

func NewReconciler(log *logrus.Entry, client client.Client, role string) *Reconciler {
	return &Reconciler{
		log:  log,
		role: role,

		checker: newClusterDNSChecker(client),

		client: client,
	}
}

// Reconcile will keep checking that the cluster has a valid DNS configuration.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.CheckerEnabled) {
		r.log.Debug("controller is disabled")
		return r.reconcileDisabled(ctx)
	}

	r.log.Debug("running")
	result, checkErr := r.checker.Check(ctx)
	condition := r.condition(checkErr, result)

	err = conditions.SetCondition(ctx, r.client, condition, r.role)
	if err != nil {
		return reconcile.Result{}, err
	}

	// We always requeue here:
	// * Either immediately (with rate limiting) based on the error
	//   when checkErr != nil.
	// * Or based on RequeueAfter when err == nil.
	return reconcile.Result{RequeueAfter: time.Hour}, checkErr
}

func (r *Reconciler) reconcileDisabled(ctx context.Context) (ctrl.Result, error) {
	condition := &operatorv1.OperatorCondition{
		Type:   arov1alpha1.DefaultClusterDNS,
		Status: operatorv1.ConditionUnknown,
	}

	return reconcile.Result{}, conditions.SetCondition(ctx, r.client, condition, r.role)
}

func (r *Reconciler) condition(checkErr error, result result) *operatorv1.OperatorCondition {
	if checkErr != nil {
		return &operatorv1.OperatorCondition{
			Type:    arov1alpha1.DefaultClusterDNS,
			Status:  operatorv1.ConditionFalse,
			Message: checkErr.Error(),
			Reason:  "CheckFailed",
		}
	}

	status := operatorv1.ConditionTrue
	if !result.success {
		status = operatorv1.ConditionFalse
	}

	return &operatorv1.OperatorCondition{
		Type:    arov1alpha1.DefaultClusterDNS,
		Status:  status,
		Message: result.message,
		Reason:  "CheckDone",
	}
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	defaultClusterDNSPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == "default"
	})

	builder := ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Watches(
			&operatorv1.DNS{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(defaultClusterDNSPredicate),
		)

	return builder.Named(ControllerName).Complete(r)
}
