package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	configv1 "github.com/openshift/api/config/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	ControllerName               = "GuardRails"
	controllerNamespace          = "aro.guardrails.namespace"
	controllerPullSpec           = "aro.guardrails.deploy.pullspec"
	controllerManagerRequestsCPU = "aro.guardrails.deploy.manager.requests.cpu"
	controllerManagerRequestsMem = "aro.guardrails.deploy.manager.requests.mem"
	controllerManagerLimitCPU    = "aro.guardrails.deploy.manager.limit.cpu"
	controllerManagerLimitMem    = "aro.guardrails.deploy.manager.limit.mem"
	controllerAuditRequestsCPU   = "aro.guardrails.deploy.audit.requests.cpu"
	controllerAuditRequestsMem   = "aro.guardrails.deploy.audit.requests.mem"
	controllerAuditLimitCPU      = "aro.guardrails.deploy.audit.limit.cpu"
	controllerAuditLimitMem      = "aro.guardrails.deploy.audit.limit.mem"

	controllerValidatingWebhookFailurePolicy = "aro.guardrails.validatingwebhook.managed"
	controllerValidatingWebhookTimeout       = "aro.guardrails.validatingwebhook.timeoutSeconds"
	controllerMutatingWebhookFailurePolicy   = "aro.guardrails.mutatingwebhook.managed"
	controllerMutatingWebhookTimeout         = "aro.guardrails.mutatingwebhook.timeoutSeconds"

	controllerReconciliationMinutes     = "aro.guardrails.reconciliationMinutes"
	controllerPolicyManagedTemplate     = "aro.guardrails.policies.%s.managed"
	controllerPolicyEnforcementTemplate = "aro.guardrails.policies.%s.enforcement"

	RoleSCCResourceName = "aro.guardrails.role.scc.resourcename"

	defaultNamespace = "openshift-azure-guardrails"

	defaultManagerRequestsCPU = "100m"
	defaultManagerLimitCPU    = "1000m"
	defaultManagerRequestsMem = "512Mi"
	defaultManagerLimitMem    = "512Mi"
	defaultAuditRequestsCPU   = "100m"
	defaultAuditLimitCPU      = "1000m"
	defaultAuditRequestsMem   = "512Mi"
	defaultAuditLimitMem      = "512Mi"

	defaultReconciliationMinutes = "60"

	defaultValidatingWebhookFailurePolicy = "Ignore"
	defaultValidatingWebhookTimeout       = "3"
	defaultMutatingWebhookFailurePolicy   = "Ignore"
	defaultMutatingWebhookTimeout         = "1"

	defaultRoleSCCResourceName411 = "restricted-v2"
	defaultRoleSCCResourceName410 = "restricted"

	gkDeploymentPath  = "staticresources"
	gkTemplatePath    = "policies/gktemplates"
	gkConstraintsPath = "policies/gkconstraints"
)

//go:embed staticresources
var staticFiles embed.FS

//go:embed policies/gktemplates
var gkPolicyTemplates embed.FS

//go:embed policies/gkconstraints
var gkPolicyConstraints embed.FS

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

	deployConfig.RoleSCCResourceName = instance.Spec.OperatorFlags.GetWithDefault(RoleSCCResourceName, defaultRoleSCCResourceName411)
	// for version 4.10 and below choose defaultRoleSCCResourceName410 scc
	if lt411, err := r.VersionLT411(ctx); err == nil && lt411 {
		deployConfig.RoleSCCResourceName = instance.Spec.OperatorFlags.GetWithDefault(RoleSCCResourceName, defaultRoleSCCResourceName410)
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

func (r *Reconciler) VersionLT411(ctx context.Context) (bool, error) {
	cv := &configv1.ClusterVersion{}
	err := r.client.Get(ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		return false, err
	}
	clusterVersion, err := version.GetClusterVersion(cv)
	if err != nil {
		r.log.Errorf("error getting the OpenShift version: %v", err)
		return false, err
	}
	ver411, _ := version.ParseVersion("4.11.0")
	return clusterVersion.Lt(ver411), nil
}

func (r *Reconciler) getGatekeeperDeployedNs(ctx context.Context, instance *arov1alpha1.Cluster) (string, error) {
	name := ""
	if r.kubernetescli == nil {
		r.log.Debug("nil kubernetescli object")
		return "", nil
	}
	start := time.Now()
	namespaces, err := r.kubernetescli.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		r.log.Warnf("Error retrieving namespaces: %v", err)
		return "", err
	}
	guardrails_namespace := instance.Spec.OperatorFlags.GetWithDefault(controllerNamespace, defaultNamespace)
	for _, ns := range namespaces.Items {
		if ns.Name == guardrails_namespace {
			// skip guardrails ns
			continue
		}
		deployments, err := r.kubernetescli.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{
			LabelSelector: "gatekeeper.sh/system=yes",
		})
		if err != nil {
			r.log.Warnf("Error retrieving deployments in namespace %s: %v", ns.Name, err)
			continue
		}
		if len(deployments.Items) > 0 {
			name = ns.Name
			break
		}
	}
	dura := time.Since(start)
	msg := "Gatekeeper not found"
	if name != "" {
		msg = "Found another gatekeeper deployed in namespace " + name
	}
	r.log.Infof("%s, search took %s.", msg, dura.String())
	return name, nil
}
