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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
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

		uns, gvk, err := unstructured.UnstructuredJSONScheme.Decode(data, nil, nil)
		if err != nil {
			return err
		}

		if managed != "true" {
			err := r.dh.EnsureDeleted(ctx, *gvk, client.ObjectKeyFromObject(uns.(client.Object)))
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

		uns, gvk, err := unstructured.UnstructuredJSONScheme.Decode(data, nil, nil)
		if err != nil {
			return err
		}
		err = r.dh.EnsureDeleted(ctx, *gvk, client.ObjectKeyFromObject(uns.(client.Object)))
		if err != nil && !kerrors.IsNotFound(err) && !strings.Contains(strings.ToLower(err.Error()), "notfound") {
			return err
		}
	}
	return nil
}

func (r *Reconciler) policyTicker(ctx context.Context, instance *arov1alpha1.Cluster) {
	r.policyTickerDone = make(chan bool)
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
		case done := <-r.policyTickerDone:
			if done {
				r.policyTickerDone = nil
				return
			}
			// false to trigger a ticker reset
			r.log.Infof("policyTicker reset to %d min", r.reconciliationMinutes)
			ticker.Reset(time.Duration(r.reconciliationMinutes) * time.Minute)
		case <-ticker.C:
			err = r.ensurePolicy(ctx, gkPolicyConstraints, gkConstraintsPath)
			if err != nil {
				r.log.Errorf("policyTicker ensurePolicy error %s", err.Error())
			}
		}
	}
}

func (r *Reconciler) startTicker(ctx context.Context, instance *arov1alpha1.Cluster) {
	minutes := instance.Spec.OperatorFlags.GetWithDefault(controllerReconciliationMinutes, defaultReconciliationMinutes)
	min, err := strconv.Atoi(minutes)
	if err != nil {
		min, _ = strconv.Atoi(defaultReconciliationMinutes)
	}
	if r.reconciliationMinutes != min && r.policyTickerDone != nil {
		// trigger ticker reset
		r.reconciliationMinutes = min
		r.policyTickerDone <- false
	}

	// make sure only one ticker started
	if r.policyTickerDone == nil {
		go r.policyTicker(ctx, instance)
	}
}

func (r *Reconciler) stopTicker() {
	if r.policyTickerDone != nil {
		r.policyTickerDone <- true
		close(r.policyTickerDone)
	}
}

func (r *Reconciler) IsConstraintTemplateReady(ctx context.Context, config interface{}) (bool, error) {
	resources, err := r.gkPolicyTemplate.Template(config)
	if err != nil {
		return false, err
	}
	for _, resource := range resources {
		gvk := resource.GetObjectKind().GroupVersionKind()
		if gvk.Group == "templates.gatekeeper.sh" && gvk.Kind == "ConstraintTemplate" {
			rname, ok := resource.(metav1.Object)
			if !ok {
				return false, fmt.Errorf("can't ")
			}

			// for the fetched object
			ct := &unstructured.Unstructured{}
			ct.SetGroupVersionKind(gvk)

			err := r.dh.GetOne(ctx, types.NamespacedName{Name: rname.GetName()}, ct)
			if err != nil {
				return false, err
			}

			ready, ok, err := unstructured.NestedBool(ct.UnstructuredContent(), "Status", "Created")
			if !ok || err != nil {
				return false, err
			}
			return ready, nil
		}
	}
	return true, nil
}
