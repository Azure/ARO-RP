package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest/fake"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

type mockGVRResolver struct{}

func (gvr mockGVRResolver) Refresh() error {
	return nil
}

func (gvr mockGVRResolver) Resolve(groupKind, optionalVersion string) (*schema.GroupVersionResource, error) {
	return &schema.GroupVersionResource{Group: "metal3.io", Version: "v1alpha1", Resource: "configmap"}, nil
}

func TestEsureDeleted(t *testing.T) {
	ctx := context.Background()

	mockGVRResolver := mockGVRResolver{}

	mockRestCLI := &fake.RESTClient{
		GroupVersion:         schema.GroupVersion{Group: "testgroup", Version: "v1"},
		NegotiatedSerializer: resource.UnstructuredPlusDefaultContentConfig().NegotiatedSerializer,
		Client: fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
			switch req.Method {
			case http.MethodDelete:
				switch req.URL.Path {
				case "/apis/metal3.io/v1alpha1/namespaces/test-ns-1/configmap/test-name-1":
					return &http.Response{StatusCode: http.StatusNotFound}, nil
				case "/apis/metal3.io/v1alpha1/namespaces/test-ns-2/configmap/test-name-2":
					return &http.Response{StatusCode: http.StatusInternalServerError}, nil
				case "/apis/metal3.io/v1alpha1/namespaces/test-ns-3/configmap/test-name-3":
					return &http.Response{StatusCode: http.StatusOK}, nil
				default:
					t.Fatalf("unexpected path: %#v\n%#v", req.URL, req)
					return nil, nil
				}
			default:
				t.Fatalf("unexpected request: %s %#v\n%#v", req.Method, req.URL, req)
				return nil, nil
			}
		}),
	}

	dh := &dynamicHelper{
		GVRResolver: mockGVRResolver,
		restcli:     mockRestCLI,
		log:         logrus.NewEntry(logrus.StandardLogger()),
	}

	err := dh.EnsureDeleted(ctx, "configmap", "test-ns-1", "test-name-1")
	if err != nil {
		t.Errorf("no error should be bounced for status not found, but got: %v", err)
	}

	err = dh.EnsureDeleted(ctx, "configmap", "test-ns-2", "test-name-2")
	if err == nil {
		t.Errorf("function should handle failure response (non-404) correctly")
	}

	err = dh.EnsureDeleted(ctx, "configmap", "test-ns-3", "test-name-3")
	if err != nil {
		t.Errorf("function should handle success response correctly")
	}
}

func TestMakeURLSegments(t *testing.T) {
	for _, tt := range []struct {
		gvr         *schema.GroupVersionResource
		namespace   string
		uname, name string
		url         []string
		want        []string
	}{
		{
			uname: "Group is empty",
			gvr: &schema.GroupVersionResource{
				Group:    "",
				Version:  "4.10",
				Resource: "test-resource",
			},
			namespace: "openshift",
			name:      "test-name-1",
			want:      []string{"api", "4.10", "namespaces", "openshift", "test-resource", "test-name-1"},
		},
		{
			uname: "Group is not empty",
			gvr: &schema.GroupVersionResource{
				Group:    "test-group",
				Version:  "4.10",
				Resource: "test-resource",
			},
			namespace: "openshift-apiserver",
			name:      "test-name-2",
			want:      []string{"apis", "test-group", "4.10", "namespaces", "openshift-apiserver", "test-resource", "test-name-2"},
		},
		{
			uname: "Namespace is empty",
			gvr: &schema.GroupVersionResource{
				Group:    "test-group",
				Version:  "4.10",
				Resource: "test-resource",
			},
			namespace: "",
			name:      "test-name-3",
			want:      []string{"apis", "test-group", "4.10", "test-resource", "test-name-3"},
		},
		{
			uname: "Namespace is not empty",
			gvr: &schema.GroupVersionResource{
				Group:    "test-group",
				Version:  "4.10",
				Resource: "test-resource",
			},
			namespace: "openshift-sdn",
			name:      "test-name-3",
			want:      []string{"apis", "test-group", "4.10", "namespaces", "openshift-sdn", "test-resource", "test-name-3"},
		},
		{
			uname: "Name is empty",
			gvr: &schema.GroupVersionResource{
				Group:    "test-group",
				Version:  "4.10",
				Resource: "test-resource",
			},
			namespace: "openshift-ns",
			name:      "",
			want:      []string{"apis", "test-group", "4.10", "namespaces", "openshift-ns", "test-resource"},
		},
	} {
		t.Run(tt.uname, func(t *testing.T) {
			got := makeURLSegments(tt.gvr, tt.namespace, tt.name)
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(cmp.Diff(got, tt.want))
			}
		})
	}
}
