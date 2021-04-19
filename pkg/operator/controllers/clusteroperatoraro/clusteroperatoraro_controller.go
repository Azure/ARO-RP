package clusteroperatoraro

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

const (
	clusterOperatorName = "aro"
)

// The default set of status change reasons.
const (
	reasonAsExpected   = "AsExpected"
	reasonInitializing = "Initializing"
)

type ClusterOperatorAROReconciler struct {
	// Replace configcli with CO client
	configcli configclient.Interface
	arocli    aroclient.Interface
	log       *logrus.Entry
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, configcli configclient.Interface) *ClusterOperatorAROReconciler {
	return &ClusterOperatorAROReconciler{
		configcli: configcli,
		arocli:    arocli,
		log:       log,
	}
}

// SetupWithManager setup our manager
func (r *ClusterOperatorAROReconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(meta metav1.Object, object runtime.Object) bool {
		return meta.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Owns(&configv1.ClusterOperator{}).
		Named(controllers.ClusterOperatorAROName).
		Complete(r)
}

func (r *ClusterOperatorAROReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	// TODO(mj): Reconcile will eventually be receiving a ctx (https://github.com/kubernetes-sigs/controller-runtime/blob/7ef2da0bc161d823f084ad21ff5f9c9bd6b0cc39/pkg/reconcile/reconcile.go#L93)
	ctx := context.TODO()

	cluster, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, retry.RetryOnConflict(retry.DefaultRetry, func() error {
		co, err := r.getOrCreateClusterOperator(ctx)
		if err != nil {
			return err
		}

		err = controllerutil.SetControllerReference(cluster, co, scheme.Scheme)
		if err != nil {
			return err
		}

		co, err = r.configcli.ConfigV1().ClusterOperators().Update(ctx, co, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		return r.setClusterOperatorStatus(ctx, co)
	})
}

func (r *ClusterOperatorAROReconciler) setClusterOperatorStatus(ctx context.Context, co *configv1.ClusterOperator) error {
	// TODO: Replace with a real conditions based on aro operator conditions
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

	for _, c := range conditions {
		v1helpers.SetStatusCondition(&co.Status.Conditions, c)
	}

	_, err := r.configcli.ConfigV1().ClusterOperators().UpdateStatus(ctx, co, metav1.UpdateOptions{})
	return err
}

func (r *ClusterOperatorAROReconciler) getOrCreateClusterOperator(ctx context.Context) (*configv1.ClusterOperator, error) {
	co, err := r.configcli.ConfigV1().ClusterOperators().Get(ctx, clusterOperatorName, metav1.GetOptions{})
	if !kerrors.IsNotFound(err) {
		return co, err
	}

	r.log.Infof("ClusterOperator does not exist, creating a new one.")
	defaultCo := r.defaultOperator()
	co, err = r.configcli.ConfigV1().ClusterOperators().Create(ctx, defaultCo, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	co.Status = defaultCo.Status

	return r.configcli.ConfigV1().ClusterOperators().UpdateStatus(ctx, co, metav1.UpdateOptions{})
}

func (r *ClusterOperatorAROReconciler) defaultOperator() *configv1.ClusterOperator {
	currentTime := metav1.Now()
	return &configv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterOperatorName,
		},
		Status: configv1.ClusterOperatorStatus{
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
