package autosizednodes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
)

type Reconciler struct {
	client client.Client

	log *logrus.Entry
}

const (
	ControllerName = "AutoSizedNodes"
	configName     = "dynamic-node"
)

func NewReconciler(log *logrus.Entry, client client.Client) *Reconciler {
	return &Reconciler{
		client: client,

		log: log,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	var aro arov1alpha1.Cluster
	var err error

	err = r.client.Get(ctx, request.NamespacedName, &aro)
	if err != nil {
		err = fmt.Errorf("unable to fetch aro cluster: %w", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.log.Infof("Config changed, autoSize: %t\n", aro.Spec.OperatorFlags.GetSimpleBoolean(operator.AutosizedNodesEnabled))

	// key is used to locate the object in the etcd
	key := types.NamespacedName{
		Name: configName,
	}

	if !aro.Spec.OperatorFlags.GetSimpleBoolean(operator.AutosizedNodesEnabled) {
		// defaults to deleting the config
		config := mcv1.KubeletConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: configName,
			},
		}
		err = r.client.Delete(ctx, &config)
		if err != nil {
			err = fmt.Errorf("could not delete KubeletConfig: %w", err)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	defaultConfig := makeConfig()

	var config mcv1.KubeletConfig
	err = r.client.Get(ctx, key, &config)
	if kerrors.IsNotFound(err) {
		// If config doesn't exist, create a new one
		err := r.client.Create(ctx, &defaultConfig, &client.CreateOptions{})
		if err != nil {
			err = fmt.Errorf("could not create KubeletConfig: %w", err)
		}
		return ctrl.Result{}, err
	}
	if err != nil {
		// If error, return it (controller-runtime will requeue for a retry)
		return ctrl.Result{}, fmt.Errorf("could not fetch KubeletConfig: %w", err)
	}

	// If already exists, update the spec
	config.Spec = defaultConfig.Spec
	err = r.client.Update(ctx, &config)
	if err != nil {
		err = fmt.Errorf("could not update KubeletConfig: %w", err)
	}
	return ctrl.Result{}, err
}

// SetupWithManager prepares the controller with info who to watch
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Controller adds ControllerManagedBy to KubeletConfit created by this controller.
	// Any changes will trigger reconcile, but only for that config.
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Owns(&mcv1.KubeletConfig{}).
		Named(ControllerName).
		Complete(r)
}

func makeConfig() mcv1.KubeletConfig {
	return mcv1.KubeletConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: configName,
		},
		Spec: mcv1.KubeletConfigSpec{
			AutoSizingReserved: to.Ptr(true),
			MachineConfigPoolSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "machineconfiguration.openshift.io/mco-built-in",
						Operator: metav1.LabelSelectorOpExists,
					},
				},
			},
		},
	}
}
