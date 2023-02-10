package autosizednodes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/ignition/v2/config/util"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

type Reconciler struct {
	client client.Client

	log *logrus.Entry
}

const (
	ControllerName = "AutoSizedNodes"

	ControllerEnabled = "aro.autosizednodes.enabled"
	configName        = "dynamic-node"
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

	r.log.Infof("Config changed, autoSize: %t\n", aro.Spec.OperatorFlags.GetSimpleBoolean(ControllerEnabled))

	// key is used to locate the object in the etcd
	key := types.NamespacedName{
		Name: configName,
	}

	if !aro.Spec.OperatorFlags.GetSimpleBoolean(ControllerEnabled) {
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
	clusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		name := o.GetName()
		return strings.EqualFold(arov1alpha1.SingletonClusterName, name)
	})

	b := ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(clusterPredicate))

		// Controller adds ControllerManagedBy to KubeletConfit created by this controller.
		// Any changes will trigger reconcile, but only for that config.
	return b.
		Named(ControllerName).
		Owns(&mcv1.KubeletConfig{}).
		Complete(r)
}

func makeConfig() mcv1.KubeletConfig {
	return mcv1.KubeletConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: configName,
		},
		Spec: mcv1.KubeletConfigSpec{
			AutoSizingReserved: util.BoolToPtr(true),
			MachineConfigPoolSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"pools.operator.machineconfiguration.openshift.io/worker": "",
				},
			},
		},
	}
}
