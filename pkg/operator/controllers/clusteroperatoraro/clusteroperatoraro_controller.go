package clusteroperatoraro

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	"github.com/openshift/library-go/pkg/operator/status"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	ControllerName = "ClusterOperatorARO"

	// Operator object name
	clusterOperatorName = "aro"
)

// The default set of status change reasons.
const (
	reasonAsExpected   = "AsExpected"
	reasonInitializing = "Initializing"
)

type Reconciler struct {
	log *logrus.Entry

	client client.Client
}

// TODO: Decide whether we actually going to make any progress on this. If not - clean up.
func NewReconciler(log *logrus.Entry, client client.Client) *Reconciler {
	return &Reconciler{
		log:    log,
		client: client,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.log.Debug("running")
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	co, err := r.getOrCreateClusterOperator(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = controllerutil.SetControllerReference(instance, co, scheme.Scheme)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.client.Update(ctx, co)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, r.setClusterOperatorStatus(ctx, co, instance)
}

func (r *Reconciler) setClusterOperatorStatus(ctx context.Context, originalClusterOperatorObj *configv1.ClusterOperator, cluster *arov1alpha1.Cluster) error {
	clusterOperatorObj := originalClusterOperatorObj.DeepCopy()

	currentTime := metav1.Now()
	conditions := []configv1.ClusterOperatorStatusCondition{
		{
			Type:               configv1.OperatorAvailable,
			Status:             configv1.ConditionTrue,
			LastTransitionTime: currentTime,
			Reason:             reasonAsExpected,
		},
		{
			Type:               configv1.OperatorProgressing,
			Status:             configv1.ConditionFalse,
			LastTransitionTime: currentTime,
			Reason:             reasonAsExpected,
		},
		{
			Type:               configv1.OperatorDegraded,
			Status:             configv1.ConditionFalse,
			LastTransitionTime: currentTime,
			Reason:             reasonAsExpected,
		},
		{
			Type:               configv1.OperatorUpgradeable,
			Status:             configv1.ConditionTrue,
			LastTransitionTime: currentTime,
			Reason:             reasonAsExpected,
		},
	}

	degradedInertia := status.MustNewInertia(2 * time.Minute).Inertia

	// todo: these guard checks can be removed once we are guaranteed to have at least one of each type of Controller Condition present on the cluster resource.
	if hasControllerConditionOfType(&cluster.Status.Conditions, operatorv1.OperatorStatusTypeAvailable) {
		v1helpers.SetStatusCondition(&clusterOperatorObj.Status.Conditions, status.UnionClusterCondition("Available", operatorv1.ConditionTrue, nil, cluster.Status.Conditions...))
	} else {
		v1helpers.SetStatusCondition(&clusterOperatorObj.Status.Conditions, conditions[0])
	}

	if hasControllerConditionOfType(&cluster.Status.Conditions, operatorv1.OperatorStatusTypeProgressing) {
		v1helpers.SetStatusCondition(&clusterOperatorObj.Status.Conditions, status.UnionClusterCondition("Progressing", operatorv1.ConditionFalse, nil, cluster.Status.Conditions...))
	} else {
		v1helpers.SetStatusCondition(&clusterOperatorObj.Status.Conditions, conditions[1])
	}

	if hasControllerConditionOfType(&cluster.Status.Conditions, operatorv1.OperatorStatusTypeDegraded) {
		v1helpers.SetStatusCondition(&clusterOperatorObj.Status.Conditions, status.UnionClusterCondition("Degraded", operatorv1.ConditionFalse, degradedInertia, cluster.Status.Conditions...))
	} else {
		v1helpers.SetStatusCondition(&clusterOperatorObj.Status.Conditions, conditions[2])
	}

	if equality.Semantic.DeepEqual(clusterOperatorObj.Status.Conditions, originalClusterOperatorObj.Status.Conditions) {
		return nil
	} else {
		return r.client.Status().Update(ctx, clusterOperatorObj)
	}
}

func (r *Reconciler) getOrCreateClusterOperator(ctx context.Context) (*configv1.ClusterOperator, error) {
	co := &configv1.ClusterOperator{}
	err := r.client.Get(ctx, types.NamespacedName{Name: clusterOperatorName}, co)
	if !kerrors.IsNotFound(err) {
		return co, err
	}

	r.log.Infof("ClusterOperator does not exist, creating a new one.")
	co = r.defaultOperator()
	err = r.client.Create(ctx, co)
	if err != nil {
		return nil, err
	}

	err = r.client.Status().Update(ctx, co)
	return co, err
}

func (r *Reconciler) defaultOperator() *configv1.ClusterOperator {
	currentTime := metav1.Now()
	return &configv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterOperatorName,
		},
		Status: configv1.ClusterOperatorStatus{
			Versions: []configv1.OperandVersion{
				{
					Name:    "operator",
					Version: version.GitCommit,
				},
			},
			Conditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:               configv1.OperatorAvailable,
					Status:             configv1.ConditionFalse,
					LastTransitionTime: currentTime,
					Reason:             reasonInitializing,
					Message:            "Operator is initializing",
				},
				{
					Type:               configv1.OperatorProgressing,
					Status:             configv1.ConditionTrue,
					LastTransitionTime: currentTime,
					Reason:             reasonInitializing,
					Message:            "Operator is initializing",
				},
				{
					Type:               configv1.OperatorDegraded,
					Status:             configv1.ConditionFalse,
					LastTransitionTime: currentTime,
					Reason:             reasonAsExpected,
				},
			},
		},
	}
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Owns(&configv1.ClusterOperator{}).
		Named(ControllerName).
		Complete(r)
}

func hasControllerConditionOfType(conditions *[]operatorv1.OperatorCondition, conditionType string) bool {
	for _, condition := range *conditions {
		if strings.HasSuffix(condition.Type, "Controller"+conditionType) {
			return true
		}
	}
	return false
}
