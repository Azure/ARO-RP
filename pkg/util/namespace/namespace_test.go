package namespace

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestFilteredOpenShiftNamespace(t *testing.T) {
	for _, tt := range []struct {
		namespace string
		want      bool
	}{
		{
			namespace: "openshift-apiserver",
			want:      true,
		},
		{
			namespace: "openshift-azure-operator",
			want:      true,
		},
		{
			namespace: "openshift-azure-logging",
			want:      true,
		},
		{
			namespace: "openshift-gitops",
			want:      false,
		},
		{
			namespace: "openshift-authentication",
			want:      false,
		},
		{
			namespace: "default",
			want:      false,
		},
		{
			namespace: "customer",
		},
	} {
		t.Run(tt.namespace, func(t *testing.T) {
			got := FilteredOpenShiftNamespace(tt.namespace)
			if tt.want != got {
				t.Error(got)
			}
		})
	}
}
