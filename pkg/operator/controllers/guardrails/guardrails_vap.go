package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

// gatekeeperCleanupNeeded returns true if there are Gatekeeper resources on the
// cluster that should be removed as part of the migration to VAP. This covers
// the upgrade scenario where Gatekeeper was deployed on a pre-4.17 cluster.
func (r *Reconciler) gatekeeperCleanupNeeded(ctx context.Context, instance *arov1alpha1.Cluster) bool {
	if r.cleanupNeeded {
		return true
	}
	if r.kubernetescli == nil {
		return false
	}
	ns := instance.Spec.OperatorFlags.GetWithDefault(controllerNamespace, defaultNamespace)
	_, err := r.kubernetescli.AppsV1().Deployments(ns).Get(ctx, "gatekeeper-audit", metav1.GetOptions{})
	return err == nil
}

// cleanupGatekeeper removes all Gatekeeper resources: constraints, constraint
// templates, and the deployment itself. It is safe to call when no Gatekeeper
// resources exist.
func (r *Reconciler) cleanupGatekeeper(ctx context.Context, instance *arov1alpha1.Cluster) error {
	r.log.Info("cleaning up Gatekeeper resources after upgrade to v4.17+")

	r.stopTicker()

	if err := r.removePolicy(ctx, gkPolicyConstraints, gkConstraintsPath); err != nil {
		r.log.Warnf("failed to remove Gatekeeper constraints: %s", err.Error())
	}

	if r.gkPolicyTemplate != nil {
		if err := r.gkPolicyTemplate.Remove(ctx, config.GuardRailsPolicyConfig{}); err != nil {
			r.log.Warnf("failed to remove Gatekeeper ConstraintTemplates: %s", err.Error())
		}
	}

	ns := instance.Spec.OperatorFlags.GetWithDefault(controllerNamespace, defaultNamespace)
	if err := r.deployer.Remove(ctx, config.GuardRailsDeploymentConfig{Namespace: ns}); err != nil {
		return fmt.Errorf("failed to remove Gatekeeper deployment: %w", err)
	}

	r.cleanupNeeded = false
	return nil
}

// vapValidationAction maps a Gatekeeper-style enforcement action to the
// equivalent VAP validationAction.
func vapValidationAction(gkEnforcement string) string {
	switch strings.ToLower(gkEnforcement) {
	case "deny":
		return "Deny"
	case "warn":
		return "Warn"
	case "dryrun":
		return "Audit"
	default:
		// default to warn
		return "Warn"
	}
}

// deployVAP creates or updates ValidatingAdmissionPolicy and
// ValidatingAdmissionPolicyBinding resources based on per-policy operator flags.
func (r *Reconciler) deployVAP(ctx context.Context, instance *arov1alpha1.Cluster) error {
	r.log.Info("reconciling VAP policies")

	entries, err := fs.ReadDir(vapPolicies, vapPolicyPath)
	if err != nil {
		return fmt.Errorf("reading VAP policy directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		policyName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))

		managed, enforcement, err := r.getPolicyConfig(ctx, instance, entry.Name())
		if err != nil {
			return err
		}

		if !strings.EqualFold(managed, "true") {
			if err := r.removeVAPPolicy(ctx, policyName); err != nil {
				r.log.Warnf("failed to remove unmanaged VAP policy %s: %s", policyName, err.Error())
			}
			continue
		}

		if err := r.ensureVAPPolicy(ctx, entry.Name()); err != nil {
			return fmt.Errorf("ensuring VAP policy %s: %w", policyName, err)
		}

		validationAction := vapValidationAction(enforcement)
		if err := r.ensureVAPBinding(ctx, policyName, validationAction); err != nil {
			return fmt.Errorf("ensuring VAP binding for %s: %w", policyName, err)
		}
	}

	return nil
}

// ensureVAPPolicy creates the ValidatingAdmissionPolicy if it does not exist.
func (r *Reconciler) ensureVAPPolicy(ctx context.Context, filename string) error {
	data, err := fs.ReadFile(vapPolicies, filepath.Join(vapPolicyPath, filename))
	if err != nil {
		return err
	}
	uns, err := dynamichelper.DecodeUnstructured(data)
	if err != nil {
		return err
	}
	return r.dh.Ensure(ctx, uns)
}

// ensureVAPBinding deletes an existing binding and recreates it so the
// validationActions field always reflects the current enforcement setting.
func (r *Reconciler) ensureVAPBinding(ctx context.Context, policyName, validationAction string) error {
	bindingFile := policyName + "-binding.yaml"
	tmpl, err := template.ParseFS(vapBindings, filepath.Join(vapBindingPath, bindingFile))
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	cfg := &config.GuardRailsVAPBindingConfig{
		ValidationAction: validationAction,
	}
	for _, t := range tmpl.Templates() {
		if err := t.Execute(buf, cfg); err != nil {
			return err
		}
	}

	uns, err := dynamichelper.DecodeUnstructured(buf.Bytes())
	if err != nil {
		return err
	}

	bindingName := uns.GetName()
	gk := uns.GroupVersionKind().GroupKind().String()
	ver := uns.GroupVersionKind().Version

	// Delete then recreate to pick up enforcement changes; ensureUnstructuredObj
	// in the dynamic helper only checks Gatekeeper enforcementAction, not VAP
	// validationActions, so an in-place update would silently no-op.
	if err := r.dh.EnsureDeletedGVR(ctx, gk, "", bindingName, ver); err != nil {
		r.log.Warnf("failed to delete existing VAP binding %s for recreation: %s", bindingName, err.Error())
	}

	return r.dh.Ensure(ctx, uns)
}

// removeVAPPolicy removes both the ValidatingAdmissionPolicyBinding and the
// ValidatingAdmissionPolicy for a given policy name.
func (r *Reconciler) removeVAPPolicy(ctx context.Context, policyName string) error {
	bindingGK := "ValidatingAdmissionPolicyBinding.admissionregistration.k8s.io"
	if err := r.dh.EnsureDeletedGVR(ctx, bindingGK, "", policyName+"-binding", "v1"); err != nil {
		r.log.Warnf("failed to remove VAP binding %s-binding: %s", policyName, err.Error())
	}

	policyGK := "ValidatingAdmissionPolicy.admissionregistration.k8s.io"
	if err := r.dh.EnsureDeletedGVR(ctx, policyGK, "", policyName, "v1"); err != nil {
		r.log.Warnf("failed to remove VAP policy %s: %s", policyName, err.Error())
	}
	return nil
}

// removeAllVAP removes every VAP policy and binding known to the controller.
func (r *Reconciler) removeAllVAP(ctx context.Context) error {
	entries, err := fs.ReadDir(vapPolicies, vapPolicyPath)
	if err != nil {
		return fmt.Errorf("reading VAP policy directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		policyName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if err := r.removeVAPPolicy(ctx, policyName); err != nil {
			r.log.Warnf("failed to remove VAP policy %s: %s", policyName, err.Error())
		}
	}
	return nil
}
