package dynamicworkaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
)

// machineConfigFromCatalog builds the MachineConfig object the reconciler will
// Ensure. catalogVersion is recorded as an annotation so an operator inspecting
// a live MachineConfig can tell which catalog publication produced it.
//
// We bypass json.Marshal on the ignition body: it's already a json.RawMessage
// the publisher gave us, so we hand it straight to MachineConfig.Spec.Config.Raw.
// This keeps the operator agnostic to ignition spec versions — MCO does that
// validation when it tries to render the config.
func machineConfigFromCatalog(w *Workaround, catalogVersion string) *mcv1.MachineConfig {
	labels := map[string]string{
		MachineConfigRoleLabel: w.Role,
		CatalogManagedByLabel:  "true",
		CatalogNameLabel:       w.Name,
	}
	annotations := map[string]string{
		"aro.openshift.io/dynamic-workaround-catalog-version": catalogVersion,
	}
	if w.Description != "" {
		annotations["aro.openshift.io/dynamic-workaround-description"] = w.Description
	}

	return &mcv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mcv1.SchemeGroupVersion.String(),
			Kind:       "MachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        w.MachineConfigName,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: mcv1.MachineConfigSpec{
			Config: runtime.RawExtension{
				// Copy the bytes so a later catalog refetch can't mutate
				// shared state through the same RawMessage backing array.
				Raw: append([]byte(nil), w.Ignition...),
			},
		},
	}
}
