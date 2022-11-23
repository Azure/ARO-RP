package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (r *Reconciler) getDefaultDeployConfig(ctx context.Context, instance *arov1alpha1.Cluster) *config.GuardRailsDeploymentConfig {
	// apply the default value if the flag is empty or missing
	deployConfig := &config.GuardRailsDeploymentConfig{
		Pullspec:  instance.Spec.OperatorFlags.GetWithDefault(controllerPullSpec, version.GateKeeperImage(instance.Spec.ACRDomain)),
		Namespace: instance.Spec.OperatorFlags.GetWithDefault(controllerNamespace, defaultNamespace),

		ManagerRequestsCPU: instance.Spec.OperatorFlags.GetWithDefault(controllerManagerRequestsCPU, defaultManagerRequestsCPU),
		ManagerLimitCPU:    instance.Spec.OperatorFlags.GetWithDefault(controllerManagerLimitCPU, defaultManagerLimitCPU),
		ManagerRequestsMem: instance.Spec.OperatorFlags.GetWithDefault(controllerManagerRequestsMem, defaultManagerRequestsMem),
		ManagerLimitMem:    instance.Spec.OperatorFlags.GetWithDefault(controllerManagerLimitMem, defaultManagerLimitMem),

		AuditRequestsCPU: instance.Spec.OperatorFlags.GetWithDefault(controllerAuditRequestsCPU, defaultAuditRequestsCPU),
		AuditLimitCPU:    instance.Spec.OperatorFlags.GetWithDefault(controllerAuditLimitCPU, defaultAuditLimitCPU),
		AuditRequestsMem: instance.Spec.OperatorFlags.GetWithDefault(controllerAuditRequestsMem, defaultAuditRequestsMem),
		AuditLimitMem:    instance.Spec.OperatorFlags.GetWithDefault(controllerAuditLimitMem, defaultAuditLimitMem),

		ValidatingWebhookTimeout:       instance.Spec.OperatorFlags.GetWithDefault(controllerValidatingWebhookTimeout, defaultValidatingWebhookTimeout),
		ValidatingWebhookFailurePolicy: instance.Spec.OperatorFlags.GetWithDefault(controllerValidatingWebhookFailurePolicy, defaultValidatingWebhookFailurePolicy),
		MutatingWebhookTimeout:         instance.Spec.OperatorFlags.GetWithDefault(controllerMutatingWebhookTimeout, defaultMutatingWebhookTimeout),
		MutatingWebhookFailurePolicy:   instance.Spec.OperatorFlags.GetWithDefault(controllerMutatingWebhookFailurePolicy, defaultMutatingWebhookFailurePolicy),
	}
	validatingManaged := instance.Spec.OperatorFlags.GetWithDefault(controllerValidatingWebhookFailurePolicy, "")
	switch {
	case validatingManaged == "":
		deployConfig.ValidatingWebhookFailurePolicy = defaultValidatingWebhookFailurePolicy
	case strings.EqualFold(validatingManaged, "true"):
		deployConfig.ValidatingWebhookFailurePolicy = "Fail"
	case strings.EqualFold(validatingManaged, "false"):
		deployConfig.ValidatingWebhookFailurePolicy = "Ignore"
	}
	mutatingManaged := instance.Spec.OperatorFlags.GetWithDefault(controllerMutatingWebhookFailurePolicy, "")
	switch {
	case mutatingManaged == "":
		deployConfig.MutatingWebhookFailurePolicy = defaultMutatingWebhookFailurePolicy
	case strings.EqualFold(mutatingManaged, "true"):
		deployConfig.MutatingWebhookFailurePolicy = "Fail"
	case strings.EqualFold(mutatingManaged, "false"):
		deployConfig.MutatingWebhookFailurePolicy = "Ignore"
	}
	r.namespace = deployConfig.Namespace
	return deployConfig
}

func (r *Reconciler) gatekeeperDeploymentIsReady(ctx context.Context, deployConfig *config.GuardRailsDeploymentConfig) (bool, error) {
	if ready, err := r.deployer.IsReady(ctx, deployConfig.Namespace, "gatekeeper-audit"); !ready || err != nil {
		return ready, err
	}
	return r.deployer.IsReady(ctx, deployConfig.Namespace, "gatekeeper-controller-manager")
}
