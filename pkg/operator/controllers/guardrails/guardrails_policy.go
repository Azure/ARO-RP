package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

func (r *Reconciler) getPolicyConfig(ctx context.Context, instance *arov1alpha1.Cluster, na string) (string, string, error) {
	parts := strings.Split(na, ".")
	if len(parts) < 1 {
		return "", "", errors.New("unrecognised name: " + na)
	}
	name := parts[0]

	managedPath := fmt.Sprintf(controllerPolicyManagedTemplate, name)
	managed := instance.Spec.OperatorFlags.GetWithDefault(managedPath, "false")

	enforcementPath := fmt.Sprintf(controllerPolicyEnforcementTemplate, name)
	enforcement := instance.Spec.OperatorFlags.GetWithDefault(enforcementPath, "dryrun")

	return managed, enforcement, nil
}

func (r *Reconciler) ensurePolicy(ctx context.Context, fs embed.FS, path string) error {
	template, err := template.ParseFS(fs, filepath.Join(path, "*"))
	if err != nil {
		return err
	}

	instance := &arov1alpha1.Cluster{}
	err = r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return err
	}

	creates := make([]kruntime.Object, 0)
	buffer := new(bytes.Buffer)
	for _, templ := range template.Templates() {
		managed, enforcement, err := r.getPolicyConfig(ctx, instance, templ.Name())
		if err != nil {
			return err
		}
		policyConfig := &config.GuardRailsPolicyConfig{
			Enforcement: enforcement,
		}
		err = templ.Execute(buffer, policyConfig)
		if err != nil {
			return err
		}
		data := buffer.Bytes()

		uns, err := dynamichelper.DecodeUnstructured(data)
		if err != nil {
			return err
		}

		if managed != "true" {
			err := r.dh.EnsureDeletedGVR(ctx, uns.GroupVersionKind().GroupKind().String(), uns.GetNamespace(), uns.GetName(), uns.GroupVersionKind().Version)
			if err != nil && !kerrors.IsNotFound(err) && !strings.Contains(strings.ToLower(err.Error()), "notfound") {
				return err
			}
			continue
		}

		creates = append(creates, uns)
	}
	err = r.dh.Ensure(ctx, creates...)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) removePolicy(ctx context.Context, fs embed.FS, path string) error {
	template, err := template.ParseFS(fs, filepath.Join(path, "*"))
	if err != nil {
		return err
	}
	buffer := new(bytes.Buffer)
	for _, templ := range template.Templates() {
		err := templ.Execute(buffer, nil)
		if err != nil {
			return err
		}
		data := buffer.Bytes()
		uns, err := dynamichelper.DecodeUnstructured(data)
		if err != nil {
			return err
		}
		err = r.dh.EnsureDeletedGVR(ctx, uns.GroupVersionKind().GroupKind().String(), uns.GetNamespace(), uns.GetName(), uns.GroupVersionKind().Version)
		if err != nil && !kerrors.IsNotFound(err) && !strings.Contains(strings.ToLower(err.Error()), "notfound") {
			return err
		}
	}
	return nil
}

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

	r.stopGKTicker()

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

func (r *Reconciler) gkTicker(ctx context.Context, instance *arov1alpha1.Cluster, done <-chan struct{}) {
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
			err = r.ensurePolicy(ctx, gkPolicyConstraints, gkConstraintsPath)
			if err != nil {
				r.log.Errorf("gkTicker ensurePolicy error %s", err.Error())
			}
		}
	}
}

func (r *Reconciler) startGKTicker(ctx context.Context, instance *arov1alpha1.Cluster) {
	minutes := instance.Spec.OperatorFlags.GetWithDefault(controllerReconciliationMinutes, defaultReconciliationMinutes)
	min, err := strconv.Atoi(minutes)
	if err != nil {
		min, _ = strconv.Atoi(defaultReconciliationMinutes)
	}

	r.tickerMu.Lock()
	defer r.tickerMu.Unlock()

	if r.gkTickerDone != nil {
		if r.reconciliationMinutes == min {
			return
		}

		close(r.gkTickerDone)
		r.gkTickerDone = nil
	}

	done := make(chan struct{})
	r.gkTickerDone = done
	r.reconciliationMinutes = min
	go r.gkTicker(ctx, instance, done)
}

func (r *Reconciler) stopGKTicker() {
	r.tickerMu.Lock()
	defer r.tickerMu.Unlock()

	if r.gkTickerDone != nil {
		close(r.gkTickerDone)
		r.gkTickerDone = nil
	}
}
