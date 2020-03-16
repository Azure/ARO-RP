package kubeactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/restmapper"
)

func TestKubeactionsFindGVR(t *testing.T) {
	tests := []struct {
		name string
		grs  []*restmapper.APIGroupResources
		kind string
		want []*schema.GroupVersionResource
	}{
		{
			name: "find one",
			grs: []*restmapper.APIGroupResources{
				{
					Group: metav1.APIGroup{
						Name:             "",
						PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1"},
					},
					VersionedResources: map[string][]metav1.APIResource{
						"v1": {
							{
								Name: "configmaps",
								Kind: "ConfigMap",
							},
						},
					},
				},
			},
			kind: "configmap",
			want: []*schema.GroupVersionResource{{Group: "", Version: "v1", Resource: "configmaps"}},
		},
		{
			name: "find best version",
			grs: []*restmapper.APIGroupResources{
				{
					Group: metav1.APIGroup{
						PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1"},
					},
					VersionedResources: map[string][]metav1.APIResource{
						"alpha": {
							{
								Name: "configmaps",
								Kind: "ConfigMap",
							},
						},
						"v1": {
							{
								Name: "configmaps",
								Kind: "ConfigMap",
							},
						},
					},
				},
			},
			kind: "configmap",
			want: []*schema.GroupVersionResource{{Group: "", Version: "v1", Resource: "configmaps"}},
		},
		{
			name: "find full group.resource",
			grs: []*restmapper.APIGroupResources{
				{
					Group: metav1.APIGroup{
						Name:             "metal3.io",
						PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1alpha1"},
					},
					VersionedResources: map[string][]metav1.APIResource{
						"v1alpha1": {
							{
								Name: "baremetalhosts",
								Kind: "BareMetalHost",
							},
						},
					},
				},
			},
			kind: "baremetalhost.metal3.io",
			want: []*schema.GroupVersionResource{{Group: "metal3.io", Version: "v1alpha1", Resource: "baremetalhosts"}},
		},
		{
			name: "no sub.resources",
			grs: []*restmapper.APIGroupResources{
				{
					Group: metav1.APIGroup{
						Name:             "metal3.io",
						PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1alpha1"},
					},
					VersionedResources: map[string][]metav1.APIResource{
						"v1alpha1": {
							{
								Name: "baremetalhosts",
								Kind: "BareMetalHost",
							},
							{
								Name: "baremetalhosts/status",
								Kind: "BareMetalHost",
							},
						},
					},
				},
			},
			kind: "baremetalhost/status",
			want: nil,
		},
		{
			name: "empty resources",
			grs:  []*restmapper.APIGroupResources{},
			kind: "configmap",
			want: nil,
		},
		{
			name: "find all kind",
			grs: []*restmapper.APIGroupResources{
				{
					Group: metav1.APIGroup{
						Name:             "metal3.io",
						PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1alpha1"},
					},
					VersionedResources: map[string][]metav1.APIResource{
						"v1alpha1": {
							{
								Name: "baremetalhosts",
								Kind: "BareMetalHost",
							},
						},
					},
				},
				{
					Group: metav1.APIGroup{
						Name:             "plastic.io",
						PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1alpha1"},
					},
					VersionedResources: map[string][]metav1.APIResource{
						"v1alpha1": {
							{
								Name: "plastichosts",
								Kind: "BareMetalHost",
							},
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
			if got := ka.findGVR(tt.grs, tt.kind); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("kubeactions.findGVR() = %v, want %v", got, tt.want)
			}
		})
	}
}
