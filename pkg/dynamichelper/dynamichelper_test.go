package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

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
			name: "empty resources",
			kind: "configmap",
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
			ka := &dynamicHelper{apiresources: tt.resources}

			got, err := ka.findGVR(tt.kind, "")
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Error(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(got)
			}
		})
	}
}
