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
		wantError string
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
			want: []*schema.GroupVersionResource{{Group: "", Version: "v1", Resource: "configmaps"}},
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
			want: []*schema.GroupVersionResource{{Group: "", Version: "v1", Resource: "configmaps"}},
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
			want: []*schema.GroupVersionResource{{Group: "metal3.io", Version: "v1alpha1", Resource: "baremetalhosts"}},
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
		},
		{
			name: "empty resources",
			kind: "configmap",
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
			want: []*schema.GroupVersionResource{
				{Group: "metal3.io", Version: "v1alpha1", Resource: "baremetalhosts"},
				{Group: "plastic.io", Version: "v1alpha1", Resource: "plastichosts"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ka := &kubeactions{}

			got, err := ka.findGVR(tt.resources, tt.kind, "")
			if err != nil && err.Error() != tt.wantError {
				t.Error(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(got)
			}
		})
	}
}
