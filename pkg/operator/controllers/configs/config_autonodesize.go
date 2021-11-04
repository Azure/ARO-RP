// Auto Node sizing tells machine-config-operator to calculate
// system reserved dynamically based on the node physical size.
// The config is applied to all worker nodes.
// More info in the doc:
// - https://docs.openshift.com/container-platform/4.8/nodes/nodes/nodes-nodes-resources-configuring.html
package configs

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/coreos/ignition/v2/config/util"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

const (
	configName = "dynamic-node"
)

type autoNodeSizeConfig struct {
}

func NewAutoNodeSizeConfig() Config {
	return &autoNodeSizeConfig{}
}

func (c *autoNodeSizeConfig) Name() string {
	return "AutoNodeSize config"
}

func (c *autoNodeSizeConfig) IsApplicable(aro arov1alpha1.Cluster, r *Reconciler, ctx context.Context) bool {
	var config mcv1.KubeletConfig
	var err error

	key := types.NamespacedName{
		Name: configName,
	}

	if aro.Spec.Features.ReconcileAutoSizedNodes {
		err = r.Get(ctx, key, &config)
		if kerrors.IsNotFound(err) {
			// config not there add
			return true
		}
	}
	return false
}

func (c *autoNodeSizeConfig) Ensure(r *Reconciler, ctx context.Context) error {
	var err error

	// create KubeletConfig and apply it in the cluster
	config := makeConfig()

	err = r.Create(ctx, config, &client.CreateOptions{})
	return err
}

func (c *autoNodeSizeConfig) Remove(r *Reconciler, ctx context.Context) error {
	// remove KubeletConfig from the cluster
	var config mcv1.KubeletConfig
	key := types.NamespacedName{
		Name: configName,
	}

	err := r.Get(ctx, key, &config)
	if err == nil && config.Spec.AutoSizingReserved != nil && *config.Spec.AutoSizingReserved {
		// the right config is there delete
		err = r.Delete(ctx, &config, &client.DeleteOptions{})
	}

	// return error only if it is not NotFound
	return client.IgnoreNotFound(err)
}

func (c *autoNodeSizeConfig) AddOwns(builder *builder.Builder) *builder.Builder {
	return builder.Owns(&mcv1.KubeletConfig{})
}

func makeConfig() *mcv1.KubeletConfig {
	return &mcv1.KubeletConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "machineconfiguration.openshift.io/v1",
			Kind:       "KubeletConfig",
		},
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
