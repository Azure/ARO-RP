package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestFindGVR(t *testing.T) {
	tests := []struct {
		name      string
		resources []*metav1.APIResourceList
		kind      string
		want      *schema.GroupVersionResource
		wantErr   error
	}{
		{
			name: "find one",
			resources: []*metav1.APIResourceList{
				{
					GroupVersion: "v1",
					APIResources: []metav1.APIResource{
						{
							Name: "configmaps",
							Kind: "ConfigMap",
						},
					},
				},
			},
			kind: "configmap",
			want: &schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"},
		},
		{
			name: "find best version",
			resources: []*metav1.APIResourceList{
				{
					GroupVersion: "v1",
					APIResources: []metav1.APIResource{
						{
							Name: "configmaps",
							Kind: "ConfigMap",
						},
					},
				},
				{
					GroupVersion: "v1beta1",
					APIResources: []metav1.APIResource{
						{
							Name: "configmaps",
							Kind: "ConfigMap",
						},
					},
				},
			},
			kind: "configmap",
			want: &schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"},
		},
		{
			name: "find full group.resource",
			resources: []*metav1.APIResourceList{
				{
					GroupVersion: "metal3.io/v1alpha1",
					APIResources: []metav1.APIResource{
						{
							Name: "baremetalhosts",
							Kind: "BareMetalHost",
						},
					},
				},
			},
			kind: "baremetalhost.metal3.io",
			want: &schema.GroupVersionResource{Group: "metal3.io", Version: "v1alpha1", Resource: "baremetalhosts"},
		},
		{
			name: "no sub.resources",
			resources: []*metav1.APIResourceList{
				{
					GroupVersion: "metal3.io/v1alpha1",
					APIResources: []metav1.APIResource{
						{
							Name: "baremetalhosts/status",
							Kind: "BareMetalHost",
						},
					},
				},
			},
			kind: "baremetalhost/status",
			wantErr: api.NewCloudError(
				http.StatusBadRequest, api.CloudErrorCodeNotFound,
				"", "The groupKind '%s' was not found.", "baremetalhost/status"),
		},
		{
			name:      "empty resources",
			resources: []*metav1.APIResourceList{},
			kind:      "configmap",
			wantErr: api.NewCloudError(
				http.StatusBadRequest, api.CloudErrorCodeNotFound,
				"", "The groupKind '%s' was not found.", "configmap"),
		},
		{
			name: "find all kinds",
			resources: []*metav1.APIResourceList{
				{
					GroupVersion: "metal3.io/v1alpha1",
					APIResources: []metav1.APIResource{
						{
							Name: "baremetalhosts",
							Kind: "BareMetalHost",
						},
					},
				},
				{
					GroupVersion: "plastic.io/v1alpha1",
					APIResources: []metav1.APIResource{
						{
							Name: "plastichosts",
							Kind: "BareMetalHost",
						},
					},
				},
			},
			kind: "baremetalhost",
			want: nil,
			wantErr: api.NewCloudError(
				http.StatusBadRequest, api.CloudErrorCodeInvalidParameter,
				"", "The groupKind '%s' matched multiple groupKinds (baremetalhost.metal3.io, baremetalhost.plastic.io).", "baremetalhost"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dh := &dynamicHelper{apiresources: tt.resources}

			got, err := dh.findGVR(tt.kind, "")
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Error(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(got)
			}
		})
	}
}

func TestEnsure(t *testing.T) {
	tests := []struct {
		name       string
		existing   []runtime.Object
		new        *unstructured.Unstructured
		wantCreate bool
		wantUpdate bool
		wantErr    string
	}{
		{
			name: "create",
			new: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "ConfigMap",
					"metadata": map[string]interface{}{
						"namespace": "openshift-azure-logging",
						"name":      "config",
					},
				},
			},
			wantCreate: true,
		},
		{
			name: "update",
			existing: []runtime.Object{
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"kind":       "ConfigMap",
						"apiVersion": "v1",
						"metadata": map[string]interface{}{
							"namespace": "openshift-azure-logging",
							"name":      "config",
						},
						"data": map[string]interface{}{
							"audit.conf": "1",
						},
					},
				},
			},
			new: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "ConfigMap",
					"apiVersion": "v1",
					"metadata": map[string]interface{}{
						"namespace": "openshift-azure-logging",
						"name":      "config",
					},
					"data": map[string]interface{}{
						"audit.conf": "2",
					},
				},
			},
			wantUpdate: true,
		},
		{
			name: "no update needed",
			existing: []runtime.Object{
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"kind":       "ConfigMap",
						"apiVersion": "v1",
						"metadata": map[string]interface{}{
							"namespace": "openshift-azure-logging",
							"name":      "config",
						},
						"data": map[string]interface{}{
							"audit.conf": "2",
						},
					},
				},
			},
			new: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "ConfigMap",
					"apiVersion": "v1",
					"metadata": map[string]interface{}{
						"namespace": "openshift-azure-logging",
						"name":      "config",
					},
					"data": map[string]interface{}{
						"audit.conf": "2",
					},
				},
			},
		},
		{
			name: "no update needed either",
			existing: []runtime.Object{
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"kind":       "ConfigMap",
						"apiVersion": "v1",
						"metadata": map[string]interface{}{
							"namespace":  "openshift-azure-logging",
							"name":       "config",
							"generation": "4", // <- should be ignored by merge
						},
						"data": map[string]interface{}{
							"audit.conf": "2",
						},
					},
				},
			},
			new: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "ConfigMap",
					"apiVersion": "v1",
					"metadata": map[string]interface{}{
						"namespace": "openshift-azure-logging",
						"name":      "config",
					},
					"data": map[string]interface{}{
						"audit.conf": "2",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var created, updated bool
			fakeDyn := fake.NewSimpleDynamicClient(runtime.NewScheme(), tt.existing...)

			fakeDyn.PrependReactor("create", "configmaps", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				created = true
				return false, nil, nil
			})

			fakeDyn.PrependReactor("update", "configmaps", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				updated = true
				return false, nil, nil
			})

			dh := &dynamicHelper{
				log: logrus.NewEntry(logrus.StandardLogger()),
				dyn: fakeDyn,
				apiresources: []*metav1.APIResourceList{
					{
						GroupVersion: "v1",
						APIResources: []metav1.APIResource{
							{
								Name: "configmaps",
								Kind: "ConfigMap",
							},
						},
					},
				},
			}

			err := dh.Ensure(context.Background(), tt.new)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if tt.wantCreate != created {
				t.Error(created)
			}
			if tt.wantUpdate != updated {
				t.Error(updated)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	for _, tt := range []struct {
		name        string
		base        *unstructured.Unstructured
		delta       *unstructured.Unstructured
		want        *unstructured.Unstructured
		wantChanged bool
	}{
		{
			name: "changed",
			base: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"creationTimestamp": "2020-01-01T00:00:00Z", // untouched
					},
					"spec": map[string]interface{}{
						"key1": "overwritten",
						"key2": "untouched",
					},
					"status": map[string]interface{}{
						"key1": "untouched",
					},
				},
			},
			delta: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"creationTimestamp": nil,
					},
					"spec": map[string]interface{}{
						"key1": "new value",
					},
					"status": map[string]interface{}{
						"key1": "ignored",
					},
				},
			},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"creationTimestamp": "2020-01-01T00:00:00Z",
					},
					"spec": map[string]interface{}{
						"key1": "new value",
						"key2": "untouched",
					},
					"status": map[string]interface{}{
						"key1": "untouched",
					},
				},
			},
			wantChanged: true,
		},
		{
			name: "no change",
			base: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"creationTimestamp": "2020-01-01T00:00:00Z", // untouched
					},
					"spec": map[string]interface{}{
						"key1": "untouched",
					},
					"status": map[string]interface{}{
						"key1": "untouched",
					},
				},
			},
			delta: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"creationTimestamp": nil,
					},
					"spec": map[string]interface{}{
						"key1": "untouched",
					},
					"status": map[string]interface{}{
						"key1": "ignored",
					},
				},
			},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"creationTimestamp": "2020-01-01T00:00:00Z",
					},
					"spec": map[string]interface{}{
						"key1": "untouched",
					},
					"status": map[string]interface{}{
						"key1": "untouched",
					},
				},
			},
		},
		{
			name: "empty status",
			base: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"creationTimestamp": "2020-01-01T00:00:00Z", // untouched
					},
					"spec": map[string]interface{}{
						"key1": "untouched",
					},
				},
			},
			delta: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"creationTimestamp": nil,
					},
					"spec": map[string]interface{}{
						"key1": "untouched",
					},
					"status": map[interface{}]interface{}{}, // this is empty
				},
			},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"creationTimestamp": "2020-01-01T00:00:00Z",
					},
					"spec": map[string]interface{}{
						"key1": "untouched",
					},
				},
			},
		},
		{
			name: "complex merge changed",
			base: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"key1": map[string]interface{}{
							"untouched": "test",
						},
					},
				},
			},
			delta: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"key1": map[string]interface{}{
							"untouched": "test",
						},
						"key2": map[string]interface{}{
							"new-value": "test2",
						},
					},
				},
			},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"key1": map[string]interface{}{
							"untouched": "test",
						},
						"key2": map[string]interface{}{
							"new-value": "test2",
						},
					},
				},
			},
			wantChanged: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, changed, _, err := merge(tt.base, tt.delta)
			if err != nil {
				t.Fatal(err)
			}

			if changed != tt.wantChanged {
				t.Error(changed)
			}

			if !reflect.DeepEqual(result, tt.want) {
				t.Error(result)
			}
		})
	}
}
