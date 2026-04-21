package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

// vapValidationAction maps a Gatekeeper-style enforcement action to the
// equivalent VAP validationAction.
func vapValidationAction(gkEnforcement string) string {
	switch strings.ToLower(gkEnforcement) {
	case "deny":
		return "Deny"
	case "warn":
		return "Warn"
	case "dryrun", "audit":
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

// ensureVAPPolicy creates or updates the ValidatingAdmissionPolicy.
// The dynamic helper's Ensure method detects native Kubernetes resources
// and uses server-side apply, so all policy fields are correctly reconciled.
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

// ensureVAPBinding creates or updates the ValidatingAdmissionPolicyBinding.
// The dynamic helper's Ensure method detects native Kubernetes resources
// and uses server-side apply, so validationActions changes are applied
// atomically without needing a delete-then-create workaround.
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

// vapTicker periodically re-applies VAP policies and bindings to prevent them from being externally deleted.
func (r *Reconciler) vapTicker(ctx context.Context, instance *arov1alpha1.Cluster, done <-chan struct{}) {
	var err error

	minutes := instance.Spec.OperatorFlags.GetWithDefault(controllerReconciliationMinutes, defaultReconciliationMinutes)
	r.reconciliationMinutes, err = strconv.Atoi(minutes)
	if err != nil {
		r.reconciliationMinutes, _ = strconv.Atoi(defaultReconciliationMinutes)
	}

	ticker := time.NewTicker(time.Duration(r.reconciliationMinutes) * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			err = r.deployVAP(ctx, instance)
			if err != nil {
				r.log.Errorf("vapTicker deployVAP error %s", err.Error())
			}
		}
	}
}

func (r *Reconciler) startVAPTicker(ctx context.Context, instance *arov1alpha1.Cluster) {
	minutes := instance.Spec.OperatorFlags.GetWithDefault(controllerReconciliationMinutes, defaultReconciliationMinutes)
	min, err := strconv.Atoi(minutes)
	if err != nil {
		min, _ = strconv.Atoi(defaultReconciliationMinutes)
	}

	r.tickerMu.Lock()
	defer r.tickerMu.Unlock()

	if r.vapTickerDone != nil {
		if r.reconciliationMinutes == min {
			return
		}

		close(r.vapTickerDone)
		r.vapTickerDone = nil
	}

	done := make(chan struct{})
	r.vapTickerDone = done
	r.reconciliationMinutes = min
	go r.vapTicker(ctx, instance, done)
}

func (r *Reconciler) stopVAPTicker() {
	r.tickerMu.Lock()
	defer r.tickerMu.Unlock()

	if r.vapTickerDone != nil {
		close(r.vapTickerDone)
		r.vapTickerDone = nil
	}
}
