package kubeactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestFindGVR(t *testing.T) {
	tests := []struct {
		name      string
		resources []*metav1.APIResourceList
		kind      string
		want      []*schema.GroupVersionResource
	}{
		{
			name: "find one",
			resources: []*metav1.APIResourceList{
				{
					APIResources: []metav1.APIResource{
						{
							Name:    "configmaps",
							Group:   "",
							Version: "v1",
							Kind:    "ConfigMap",
						},
					},
				},
			},
			kind: "configmap",
			want: []*schema.GroupVersionResource{{Group: "", Version: "v1", Resource: "configmaps"}},
		},
		{
			name: "find best version",
			resources: []*metav1.APIResourceList{
				{
					APIResources: []metav1.APIResource{
						{
							Name:    "configmaps",
							Group:   "",
							Version: "v1",
							Kind:    "ConfigMap",
						},
					},
				},
				{
					APIResources: []metav1.APIResource{
						{
							Name:    "configmaps",
							Group:   "",
							Version: "v1beta1",
							Kind:    "ConfigMap",
						},
					},
				},
			},
			kind: "configmap",
			want: []*schema.GroupVersionResource{{Group: "", Version: "v1", Resource: "configmaps"}},
		},
		{
			name: "find full group.resource",
			resources: []*metav1.APIResourceList{
				{
					APIResources: []metav1.APIResource{
						{
							Name:    "baremetalhosts",
							Group:   "metal3.io",
							Version: "v1alpha1",
							Kind:    "BareMetalHost",
						},
					},
				},
			},
			kind: "baremetalhost.metal3.io",
			want: []*schema.GroupVersionResource{{Group: "metal3.io", Version: "v1alpha1", Resource: "baremetalhosts"}},
		},
		{
			name: "no sub.resources",
			resources: []*metav1.APIResourceList{
				{
					APIResources: []metav1.APIResource{
						{
							Name:    "baremetalhosts/status",
							Group:   "metal3.io",
							Version: "v1alpha1",
							Kind:    "BareMetalHost",
						},
					},
				},
			},
			kind: "baremetalhost/status",
		},
		{
			name: "empty resources",
			kind: "configmap",
		},
		{
			name: "find all kinds",
			resources: []*metav1.APIResourceList{
				{
					APIResources: []metav1.APIResource{
						{
							Name:    "baremetalhosts",
							Group:   "metal3.io",
							Version: "v1alpha1",
							Kind:    "BareMetalHost",
						},
					},
				},
				{
					APIResources: []metav1.APIResource{
						{
							Name:    "plastichosts",
							Group:   "plastic.io",
							Version: "v1alpha1",
							Kind:    "BareMetalHost",
						},
					},
				},
			},
			kind: "baremetalhost",
			want: []*schema.GroupVersionResource{
				{Group: "metal3.io", Version: "v1alpha1", Resource: "baremetalhosts"},
				{Group: "plastic.io", Version: "v1alpha1", Resource: "plastichosts"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ka := &kubeactions{}

			got := ka.findGVR(tt.resources, tt.kind, "")
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(got)
			}
		})
	}
}
