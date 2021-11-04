package configs

// Config represents single configuration that will change in the cluster
// It can influence multiple objects and perform complex checks to apply
// conditional configuration
import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/builder"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Config represents single config to apply in the cluster
type Config interface {
	Name() string

	// IsApplicable to the cluster (config)
	IsApplicable(arov1alpha1.Cluster, *Reconciler, context.Context) bool

	// Ensure applies the config to the cluster
	Ensure(*Reconciler, context.Context) error

	// Remove the config from the cluster
	Remove(*Reconciler, context.Context) error

	// AddOwns calls builder.Owns and append the cluster Object the this config
	// creates, edit or deletes
	AddOwns(*builder.Builder) *builder.Builder
}
