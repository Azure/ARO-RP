package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"testing"
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
