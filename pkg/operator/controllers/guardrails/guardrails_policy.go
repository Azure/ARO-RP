package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
)

func (r *Reconciler) ensurePolicies(ctx context.Context, instance *arov1alpha1.Cluster) error {
	templates, err := r.gkPolicyTemplate.Templates()
	if err != nil {
		return err
	}

	managed := instance.Spec.OperatorFlags.GetWithDefault(controllerManaged, "")

	creates := make([]kruntime.Object, 0)
	deletes := make([]kruntime.Object, 0)
	for _, templ := range templates {
		policyConfig, err := config.GetPolicyConfig(instance, templ.Name())
		if err != nil {
			return err
		}
		buffer := new(bytes.Buffer)
		err = templ.Execute(buffer, policyConfig)
		if err != nil {
			return err
		}
		data := buffer.Bytes()

		json, err := yaml.YAMLToJSON(data)
		if err != nil {
			return err
		}

		uns, _, err := unstructured.UnstructuredJSONScheme.Decode(json, nil, nil)
		if err != nil {
			return err
		}

		if policyConfig.Managed == "true" || managed == "false" {
			creates = append(creates, uns)
		} else {
			deletes = append(deletes, uns)
		}
	}

	// Delete the ones we don't want to keep first, but be tolerant of errors
	// until we're done
	var deleteError error
	for _, toDelete := range deletes {
		deleteError = r.dh.EnsureObjectDeleted(ctx, toDelete)
		if deleteError != nil {
			r.log.Errorf("error deleting policy: %v", deleteError)
		}
	}
	if deleteError != nil {
		return deleteError
	}

	err = r.dh.Ensure(ctx, creates...)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) IsConstraintTemplateReady(ctx context.Context, config interface{}) (bool, error) {
	resources, err := r.gkPolicyTemplate.Render(config)
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
