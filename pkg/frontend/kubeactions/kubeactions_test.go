package kubeactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/restmapper"
	clienttesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/env"
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

func newUnstructured(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
				"uid":       "some-UID-value",
			},
		},
	}
}

func TestKubeactionsCreateOrUpdateOne(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
	_env := &env.Test{}
	ctx := context.Background()
	tests := []struct {
		name            string
		grs             []*restmapper.APIGroupResources
		fakeClient      func(in string) *fake.FakeDynamicClient
		input           string
		validateActions func(t *testing.T, actions []clienttesting.Action)
		wantErr         bool
	}{
		{
			name: "create",
			input: `{
				"kind": "ConfigMap",
				"apiVersion": "v1",
				"metadata": {
					"name": "example-configmap",
					"namespace": "default"
				},
				"data": {
					"keys": "a=771 \nb=42"
				}
			}`,
			fakeClient: func(in string) *fake.FakeDynamicClient {
				return fake.NewSimpleDynamicClient(runtime.NewScheme())
			},
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				if len(actions) != 2 {
					t.Errorf("should be 2 action")
				}
				if !actions[0].Matches("get", "configmaps") {
					t.Error(spew.Sdump(actions))
				}
				if !actions[1].Matches("create", "configmaps") {
					t.Error(spew.Sdump(actions))
				}
			},
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
		},
		{
			name: "update",
			input: `{
				"kind": "ConfigMap",
				"apiVersion": "v1",
				"metadata": {
					"name": "example-configmap",
					"namespace": "default"
				},
				"data": {
					"keys": "a=771 \nb=42"
				}
			}`,
			fakeClient: func(in string) *fake.FakeDynamicClient {
				un := &unstructured.Unstructured{}
				un.UnmarshalJSON([]byte(in))
				un.SetLabels(map[string]string{"foo": "fee"})
				return fake.NewSimpleDynamicClient(runtime.NewScheme(), un)
			},
			validateActions: func(t *testing.T, actions []clienttesting.Action) {
				if len(actions) != 2 {
					t.Errorf("should be 2 action")
				}
				if !actions[0].Matches("get", "configmaps") {
					t.Error(spew.Sdump(actions))
				}
				if !actions[1].Matches("update", "configmaps") {
					t.Error(spew.Sdump(actions))
				}
			},
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ka := &kubeactions{
				log: log,
				env: _env,
			}
			un := &unstructured.Unstructured{}
			err := un.UnmarshalJSON([]byte(tt.input))
			if err != nil {
				t.Errorf("kubeactions unmarshal error = %v", err)
			}
			fc := tt.fakeClient(tt.input)
			if err := ka.createOrUpdateOne(ctx, fc, tt.grs, un); (err != nil) != tt.wantErr {
				t.Errorf("kubeactions.createOrUpdateOne() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.validateActions(t, fc.Actions())
		})
	}
}
