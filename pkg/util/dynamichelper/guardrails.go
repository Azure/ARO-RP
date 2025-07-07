package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

// the UnstructuredObj related stuff is specifically for the Guardrails
// to handle the gatekeeper Constraint as it does not have a scheme that can be imported
func (dh *dynamicHelper) ensureUnstructuredObj(ctx context.Context, uns *unstructured.Unstructured) error {
	gvr, err := dh.Resolve(uns.GroupVersionKind().GroupKind().String(), uns.GroupVersionKind().Version)
	if err != nil {
		return err
	}

	create := false
	obj, err := dh.dynamicClient.Resource(*gvr).Namespace(uns.GetNamespace()).Get(ctx, uns.GetName(), metav1.GetOptions{})
	if err != nil {
		if !notFound(err) {
			return err
		}
		create = true
	}
	if create {
		dh.log.Infof("Create %s", keyFunc(uns.GroupVersionKind().GroupKind(), uns.GetNamespace(), uns.GetName()))
		if _, err = dh.dynamicClient.Resource(*gvr).Namespace(uns.GetNamespace()).Create(ctx, uns, metav1.CreateOptions{}); err != nil {
			return err
		}
		return nil
	}
	enNew, err := GetEnforcementAction(uns)
	if err != nil {
		return nil
	}
	enOld, err := GetEnforcementAction(obj)
	if err != nil {
		return nil
	}
	if strings.EqualFold(enOld, enNew) {
		// currently EnforcementAction is the only part that may change in an update
		return nil
	}
	dh.log.Infof("Update %s: enforcementAction: %s->%s", keyFunc(uns.GroupVersionKind().GroupKind(), uns.GetNamespace(), uns.GetName()), enOld, enNew)
	uns.SetResourceVersion(obj.GetResourceVersion())

	if _, err = dh.dynamicClient.Resource(*gvr).Namespace(uns.GetNamespace()).Update(ctx, uns, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

func GetEnforcementAction(obj *unstructured.Unstructured) (string, error) {
	name := obj.GetName()
	ns := obj.GetNamespace()
	field, ok := obj.Object["spec"]
	if !ok {
		return "", fmt.Errorf("%s/%s: get spec failed", ns, name)
	}
	spec, ok := field.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("%s/%s: spec: %T is not map", ns, name, field)
	}

	field, ok = spec["enforcementAction"]
	if !ok {
		return "", fmt.Errorf("%s/%s: get enforcementAction failed", ns, name)
	}
	enforce, ok := field.(string)
	if !ok {
		return "", fmt.Errorf("%s/%s: enforcementAction: %T is not string", ns, name, field)
	}

	return enforce, nil
}

func (dh *dynamicHelper) deleteUnstructuredObj(ctx context.Context, groupKind, namespace, name string) error {
	gvr, err := dh.Resolve(groupKind, "")
	if err != nil {
		return err
	}
	uns, err := dh.dynamicClient.Resource(*gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return nil
	}
	if err != nil || uns == nil {
		return err
	}
	if err = dh.dynamicClient.Resource(*gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !notFound(err) {
		return err
	}
	return nil
}

func notFound(err error) bool {
	if err == nil || kerrors.IsNotFound(err) || strings.Contains(strings.ToLower(err.Error()), "notfound") {
		return true
	}
	return false
}

// mergeGK takes the existing (old) and desired (new) objects. It checks the
// the interested fields in the *new* object to see if an update is necessary,
// fixes up the *old* object if needed, and returns the difference for
// debugging purposes. The reason for using *old* as basis is that the *old*
// object are changed by gatekeeper binaries and the changes must be kept.
func mergeGK(old, new kruntime.Object) (kruntime.Object, bool, string, error) {
	if reflect.TypeOf(old) != reflect.TypeOf(new) {
		return nil, false, "", fmt.Errorf("types differ: %T %T", old, new)
	}
	expected := old.DeepCopyObject()

	// 1. Set defaults on old.  This gets rid of many false positive diffs.
	scheme.Scheme.Default(expected)

	// 2. Do fix-ups on a per-Kind basis.
	hasChanged := false
	switch new.(type) {
	case *appsv1.Deployment:
		hasChanged = handleDeployment(new, expected)
	case *admissionregistrationv1.ValidatingWebhookConfiguration:
		hasChanged = handleValidatingWebhook(new, expected)
	case *admissionregistrationv1.MutatingWebhookConfiguration:
		hasChanged = handleMutatingWebhook(new, expected)
	}
	var diff string
	if _, ok := expected.(*corev1.Secret); !ok { // Don't show a diff if kind is Secret
		diff = cmp.Diff(old, expected)
	}
	return expected, hasChanged, diff, nil
}

func handleDeployment(new, expected kruntime.Object) bool {
	hasChanged := false
	newDeployment, expectedDeployment := new.(*appsv1.Deployment), expected.(*appsv1.Deployment)
	for i := range expectedDeployment.Spec.Template.Spec.Containers {
		ec := &expectedDeployment.Spec.Template.Spec.Containers[i]
		nc := &newDeployment.Spec.Template.Spec.Containers[i]
		if ec.Image != nc.Image {
			ec.Image = nc.Image
			hasChanged = true
		}
		if cmpAndCopy(&nc.Resources.Limits, &ec.Resources.Limits) {
			hasChanged = true
		}
		if cmpAndCopy(&nc.Resources.Requests, &ec.Resources.Requests) {
			hasChanged = true
		}
	}
	return hasChanged
}

func handleValidatingWebhook(new, expected kruntime.Object) bool {
	hasChanged := false
	newWebhook := new.(*admissionregistrationv1.ValidatingWebhookConfiguration)
	expectedWebhook := expected.(*admissionregistrationv1.ValidatingWebhookConfiguration)
	for i := range expectedWebhook.Webhooks {
		if expectedWebhook.Webhooks[i].FailurePolicy != nil &&
			newWebhook.Webhooks[i].FailurePolicy != nil &&
			*expectedWebhook.Webhooks[i].FailurePolicy != *newWebhook.Webhooks[i].FailurePolicy {
			*expectedWebhook.Webhooks[i].FailurePolicy = *newWebhook.Webhooks[i].FailurePolicy
			hasChanged = true
		}
		if expectedWebhook.Webhooks[i].TimeoutSeconds != nil &&
			newWebhook.Webhooks[i].TimeoutSeconds != nil &&
			*expectedWebhook.Webhooks[i].TimeoutSeconds != *newWebhook.Webhooks[i].TimeoutSeconds {
			*expectedWebhook.Webhooks[i].TimeoutSeconds = *newWebhook.Webhooks[i].TimeoutSeconds
			hasChanged = true
		}
	}
	return hasChanged
}

func handleMutatingWebhook(new, expected kruntime.Object) bool {
	hasChanged := false
	newWebhook := new.(*admissionregistrationv1.MutatingWebhookConfiguration)
	expectedWebhook := expected.(*admissionregistrationv1.MutatingWebhookConfiguration)
	for i := range expectedWebhook.Webhooks {
		if expectedWebhook.Webhooks[i].FailurePolicy != nil &&
			newWebhook.Webhooks[i].FailurePolicy != nil &&
			*expectedWebhook.Webhooks[i].FailurePolicy != *newWebhook.Webhooks[i].FailurePolicy {
			*expectedWebhook.Webhooks[i].FailurePolicy = *newWebhook.Webhooks[i].FailurePolicy
			hasChanged = true
		}
		if expectedWebhook.Webhooks[i].TimeoutSeconds != nil &&
			newWebhook.Webhooks[i].TimeoutSeconds != nil &&
			*expectedWebhook.Webhooks[i].TimeoutSeconds != *newWebhook.Webhooks[i].TimeoutSeconds {
			*expectedWebhook.Webhooks[i].TimeoutSeconds = *newWebhook.Webhooks[i].TimeoutSeconds
			hasChanged = true
		}
	}
	return hasChanged
}

func cmpAndCopy(srcPtr, dstPtr *corev1.ResourceList) bool {
	src, dst := *srcPtr, *dstPtr
	hasChanged := false
	for key, val := range dst {
		if !val.Equal(src[key]) {
			dst[key] = src[key].DeepCopy()
			hasChanged = true
		}
	}
	return hasChanged
}

func (dh *dynamicHelper) IsConstraintTemplateReady(ctx context.Context, name string) (bool, error) {
	gvr := schema.GroupVersionResource{Group: "templates.gatekeeper.sh", Version: "v1beta1", Resource: "constrainttemplates"}
	ct, err := dh.dynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	status, ok := ct.Object["status"].(map[string]interface{})
	if !ok {
		return false, nil
	}
	created, ok := status["created"].(bool)
	if !ok {
		return false, nil
	}
	return created, nil
}
