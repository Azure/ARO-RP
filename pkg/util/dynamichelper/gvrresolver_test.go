package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/restmapper"
)

func TestFindGVR(t *testing.T) {
	tests := []struct {
		name      string
		resources []*restmapper.APIGroupResources
		kind      string
		want      *schema.GroupVersionResource
		wantErr   error
	}{
		{
			name: "find one",
			resources: []*restmapper.APIGroupResources{
				{
					Group: metav1.APIGroup{
						Name: "",
						Versions: []metav1.GroupVersionForDiscovery{
							{
								GroupVersion: "v1",
								Version:      "v1",
							},
						},
					},
					VersionedResources: map[string][]metav1.APIResource{
						"v1": {
							{
								Name:         "configmaps",
								SingularName: "configmap",
								Kind:         "ConfigMap",
							},
						},
					},
				},
			},
			kind: "configmap",
			want: &schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"},
		},
		{
			name: "find best version",
			resources: []*restmapper.APIGroupResources{
				{
					Group: metav1.APIGroup{
						Name: "",
						Versions: []metav1.GroupVersionForDiscovery{
							{
								GroupVersion: "v1",
								Version:      "v1",
							},
							{
								GroupVersion: "v1beta1",
								Version:      "v1beta1",
							},
						},
					},
					VersionedResources: map[string][]metav1.APIResource{
						"v1": {
							{
								Name:         "configmaps",
								SingularName: "configmap",
								Kind:         "ConfigMap",
							},
						},
						"v1beta1": {
							{
								Name:         "configmaps",
								SingularName: "configmap",
								Kind:         "ConfigMap",
							},
						},
					},
				},
			},
			kind: "configmap",
			want: &schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"},
		},
		{
			name: "find full group.resource",
			resources: []*restmapper.APIGroupResources{
				{
					Group: metav1.APIGroup{
						Name: "metal3.io",
						Versions: []metav1.GroupVersionForDiscovery{
							{
								GroupVersion: "metal3.io/v1alpha1",
								Version:      "v1alpha1",
							},
						},
					},
					VersionedResources: map[string][]metav1.APIResource{
						"v1alpha1": {
							{
								Name:         "baremetalhosts",
								SingularName: "baremetalhost",
								Kind:         "BareMetalHost",
							},
						},
					},
				},
			},
			kind: "baremetalhost.metal3.io",
			want: &schema.GroupVersionResource{Group: "metal3.io", Version: "v1alpha1", Resource: "baremetalhosts"},
		},
		{
			name: "no sub.resources",
			resources: []*restmapper.APIGroupResources{
				{
					Group: metav1.APIGroup{
						Name: "metal3.io",
						Versions: []metav1.GroupVersionForDiscovery{
							{
								GroupVersion: "metal3.io/v1alpha1",
								Version:      "v1alpha1",
							},
						},
					},
					VersionedResources: map[string][]metav1.APIResource{
						"v1alpha1": {
							{
								Name:         "baremetalhosts",
								SingularName: "baremetalhost",
								Kind:         "BareMetalHost",
							},
						},
					},
				},
			},
			kind:    "baremetalhost/status",
			wantErr: &meta.NoResourceMatchError{PartialResource: schema.GroupVersionResource{Resource: "baremetalhost/status"}},
		},
		{
			name:      "empty resources",
			resources: []*restmapper.APIGroupResources{},
			kind:      "configmap",
			wantErr:   &meta.NoResourceMatchError{PartialResource: schema.GroupVersionResource{Resource: "configmap"}},
		},
		{
			name: "find all kinds",
			resources: []*restmapper.APIGroupResources{
				{
					Group: metav1.APIGroup{
						Name: "metal3.io",
						Versions: []metav1.GroupVersionForDiscovery{
							{
								GroupVersion: "metal3.io/v1alpha1",
								Version:      "v1alpha1",
							},
						},
					},
					VersionedResources: map[string][]metav1.APIResource{
						"v1alpha1": {
							{
								Name:         "baremetalhosts",
								SingularName: "baremetalhost",
								Kind:         "BareMetalHost",
							},
						},
					},
				},
				{
					Group: metav1.APIGroup{
						Name: "plastic.io",
						Versions: []metav1.GroupVersionForDiscovery{
							{
								GroupVersion: "plastic.io/v1alpha1",
								Version:      "v1alpha1",
							},
						},
					},
					VersionedResources: map[string][]metav1.APIResource{
						"v1alpha1": {
							{
								Name:         "baremetalhosts",
								SingularName: "baremetalhost",
								Kind:         "BareMetalHost",
							},
						},
					},
				},
			},
			kind: "baremetalhost",
			want: &schema.GroupVersionResource{Group: "metal3.io", Version: "v1alpha1", Resource: "baremetalhosts"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &gvrResolver{apiresources: tt.resources}

			got, err := r.Resolve(tt.kind, "")
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("got: %#v, expected: %#v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got: %#v, expected: %#v", got, tt.want)
			}
		})
	}
}
