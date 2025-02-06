package clusteroperatoraro

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configv1helpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	"github.com/openshift/library-go/pkg/operator/status"
	operatorv1helpers "github.com/openshift/library-go/pkg/operator/v1helpers"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
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

	configv1helpers.SetStatusCondition(&clusterOperatorObj.Status.Conditions, status.UnionClusterCondition("Available", operatorv1.ConditionTrue, nil, cluster.Status.Conditions...))
	configv1helpers.SetStatusCondition(&clusterOperatorObj.Status.Conditions, status.UnionClusterCondition("Progressing", operatorv1.ConditionFalse, nil, cluster.Status.Conditions...))

	// We always set the Degraded status to false, as the operator being in Degraded state will prevent cluster upgrade.
	configv1helpers.SetStatusCondition(&clusterOperatorObj.Status.Conditions, configv1.ClusterOperatorStatusCondition{
		Type:               configv1.OperatorDegraded,
		Status:             configv1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             reasonAsExpected,
	})

	operatorv1helpers.SetOperandVersion(&clusterOperatorObj.Status.Versions, configv1.OperandVersion{Name: "operator", Version: version.GitCommit})

	if equality.Semantic.DeepEqual(clusterOperatorObj, originalClusterOperatorObj) {
		return nil
	}

	return r.client.Status().Update(ctx, clusterOperatorObj)
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
	return ctrl.NewControllerManagedBy(mgr).
		// we want to reconcile on status changes on the ARO Cluster resource here, unlike most other reconcilers
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicates.AROCluster)).
		Owns(&configv1.ClusterOperator{}).
		Named(ControllerName).
		Complete(r)
}
