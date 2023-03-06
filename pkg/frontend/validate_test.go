package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestValidateAdminKubernetesPodLogs(t *testing.T) {
	longName := strings.Repeat("x", 256)

	for _, tt := range []struct {
		test          string
		containerName string
		namespace     string
		name          string
		wantErr       string
	}{
		{
			test:          "valid openshift namespace",
			namespace:     "openshift",
			containerName: "container-01",
			name:          "Valid-NAME-01",
		},
		{
			test:          "customer namespace",
			namespace:     "customer",
			name:          "Valid-NAME-01",
			containerName: "container-01",
			wantErr:       "403: Forbidden: : Access to the provided namespace 'customer' is forbidden.",
		},
		{
			test:          "invalid namespace",
			namespace:     "openshift-/",
			name:          "Valid-NAME-01",
			containerName: "container-01",
			wantErr:       "400: InvalidParameter: : The provided namespace 'openshift-/' is invalid.",
		},
		{
			test:          "invalid name",
			namespace:     "openshift-image-registry",
			name:          longName,
			containerName: "container-01",
			wantErr:       "400: InvalidParameter: : The provided pod name '" + longName + "' is invalid.",
		},
		{
			test:          "empty name",
			namespace:     "openshift-image-registry",
			containerName: "container-01",
			wantErr:       "400: InvalidParameter: : The provided pod name '' is invalid.",
		},
		{
			test:      "empty container name",
			namespace: "openshift-image-registry",
			name:      "pod-name",
			wantErr:   "400: InvalidParameter: : The provided container name '' is invalid.",
		},
		{
			test:          "empty namespace",
			containerName: "container-01",
			name:          "pod-name",
			wantErr:       "400: InvalidParameter: : The provided namespace '' is invalid.",
		},
		{
			test:          "valid container name",
			containerName: "container-01",
			name:          "Valid-NAME-01",
			namespace:     "openshift-image-registry",
		},
		{
			test:          "valid name",
			containerName: "container-01",
			namespace:     "openshift-image-registry",
			name:          "Valid-NAME-01",
		},
		{
			test:          "invalid container name",
			containerName: "container_invalid",
			namespace:     "openshift-image-registry",
			name:          "Valid-pod-name-01",
			wantErr:       "400: InvalidParameter: : The provided container name 'container_invalid' is invalid.",
		},
	} {
		t.Run(tt.test, func(t *testing.T) {
			err := validateAdminKubernetesPodLogs(tt.namespace, tt.name, tt.containerName)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestValidateAdminKubernetesObjectsNonCustomer(t *testing.T) {
	longName := strings.Repeat("x", 256)

	for _, tt := range []struct {
		test      string
		method    string
		gvr       *schema.GroupVersionResource
		namespace string
		name      string
		wantErr   string
	}{
		{
			test:      "valid openshift namespace",
			gvr:       &schema.GroupVersionResource{Group: "openshift.io", Resource: "validkind"},
			namespace: "openshift",
			name:      "Valid-NAME-01",
		},
		{
			test:      "invalid customer namespace",
			gvr:       &schema.GroupVersionResource{Group: "openshift.io", Resource: "validkind"},
			namespace: "customer",
			name:      "Valid-NAME-01",
			wantErr:   "403: Forbidden: : Access to the provided namespace 'customer' is forbidden.",
		},
		{
			test:      "forbidden groupKind",
			gvr:       &schema.GroupVersionResource{Resource: "secrets"},
			namespace: "openshift",
			name:      "Valid-NAME-01",
			wantErr:   "403: Forbidden: : Access to secrets is forbidden.",
		},
		{
			test:      "forbidden groupKind",
			gvr:       &schema.GroupVersionResource{Group: "oauth.openshift.io", Resource: "anything"},
			namespace: "openshift",
			name:      "Valid-NAME-01",
			wantErr:   "403: Forbidden: : Access to secrets is forbidden.",
		},
		{
			test: "allowed groupKind on read",
			gvr:  &schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Resource: "clusterroles"},
			name: "Valid-NAME-01",
		},
		{
			test: "allowed groupKind on read 2",
			gvr:  &schema.GroupVersionResource{Group: "authorization.openshift.io", Resource: "clusterroles"},
			name: "Valid-NAME-01",
		},
		{
			test: "allowed groupKind on read 3",
			gvr:  &schema.GroupVersionResource{Resource: "nodes"},
			name: "Valid-NAME-01",
		},
		{
			test:    "forbidden clusterwide groupKind on read",
			gvr:     &schema.GroupVersionResource{Resource: "namespaces"},
			name:    "Valid-NAME-01",
			wantErr: "403: Forbidden: : Access to cluster-scoped object '/, Resource=namespaces' is forbidden.",
		},
		{
			test:    "forbidden clusterwide groupKind on read 2",
			gvr:     &schema.GroupVersionResource{Group: "user.openshift.io", Resource: "users"},
			name:    "Valid-NAME-01",
			wantErr: "403: Forbidden: : Access to cluster-scoped object 'user.openshift.io/, Resource=users' is forbidden.",
		},
		{
			test:    "forbidden groupKind on write",
			method:  http.MethodPost,
			gvr:     &schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Resource: "clusterroles"},
			name:    "Valid-NAME-01",
			wantErr: "403: Forbidden: : Write access to RBAC is forbidden.",
		},
		{
			test:    "forbidden groupKind on write 2",
			method:  http.MethodPost,
			gvr:     &schema.GroupVersionResource{Group: "authorization.openshift.io", Resource: "clusterroles"},
			name:    "Valid-NAME-01",
			wantErr: "403: Forbidden: : Write access to RBAC is forbidden.",
		},
		{
			test:      "empty groupKind",
			namespace: "openshift",
			name:      "Valid-NAME-01",
			wantErr:   "400: InvalidParameter: : The provided resource is invalid.",
		},
		{
			test:      "invalid namespace",
			gvr:       &schema.GroupVersionResource{Group: "openshift.io", Resource: "validkind"},
			namespace: "openshift-/",
			name:      "Valid-NAME-01",
			wantErr:   "403: Forbidden: : Access to the provided namespace 'openshift-/' is forbidden.",
		},
		{
			test:      "invalid name",
			gvr:       &schema.GroupVersionResource{Group: "openshift.io", Resource: "validkind"},
			namespace: "openshift",
			name:      longName,
			wantErr:   "400: InvalidParameter: : The provided name '" + longName + "' is invalid.",
		},
		{
			test:      "post: empty name",
			method:    http.MethodPost,
			gvr:       &schema.GroupVersionResource{Group: "openshift.io", Resource: "validkind"},
			namespace: "openshift",
			wantErr:   "400: InvalidParameter: : The provided name '' is invalid.",
		},
		{
			test:      "delete: empty name",
			method:    http.MethodDelete,
			gvr:       &schema.GroupVersionResource{Group: "openshift.io", Resource: "validkind"},
			namespace: "openshift",
			wantErr:   "400: InvalidParameter: : The provided name '' is invalid.",
		},
	} {
		t.Run(tt.test, func(t *testing.T) {
			if tt.method == "" {
				tt.method = http.MethodGet
			}

			err := validateAdminKubernetesObjectsNonCustomer(tt.method, tt.gvr, tt.namespace, tt.name)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestValidateAdminMasterVMSize(t *testing.T) {
	for _, tt := range []struct {
		test    string
		vmSize  string
		wantErr string
	}{
		{
			test:    "size is supported as master",
			vmSize:  "Standard_D8s_v3",
			wantErr: "",
		},
		{
			test:    "size is supported as master, lowercase",
			vmSize:  "standard_d8s_v3",
			wantErr: "",
		},
		{
			test:    "size is unsupported as master",
			vmSize:  "Silly_D8s_v10",
			wantErr: "400: InvalidParameter: : The provided vmSize 'Silly_D8s_v10' is unsupported for master.",
		},
		{
			test:    "size is unsupported as master, lowercase",
			vmSize:  "silly_d8s_v10",
			wantErr: "400: InvalidParameter: : The provided vmSize 'silly_d8s_v10' is unsupported for master.",
		},
	} {
		t.Run(tt.test, func(t *testing.T) {
			err := validateAdminMasterVMSize(tt.vmSize)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
