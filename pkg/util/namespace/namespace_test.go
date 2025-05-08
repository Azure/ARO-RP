package namespace

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestIsOpenShiftNamespace(t *testing.T) {
	for _, tt := range []struct {
		namespace string
		want      bool
	}{
		{
			namespace: "",
			want:      true,
		},
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
			want:      false,
		},
		{
			namespace: "openshift-/",
			want:      false,
		},
		{
			namespace: "openshift-ns",
			want:      false,
		},
		{
			namespace: "kube-ns",
			want:      false,
		},
		{
			namespace: "openshift-operator-lifecycle-manager",
			want:      true,
		},
		{
			namespace: "openshift-ovn-kubernetes",
			want:      true,
		},
		{
			namespace: "openshift-cluster-version",
			want:      true,
		},
		{
			namespace: "openshift-azure-guardrails",
			want:      true,
		},
	} {
		t.Run(tt.namespace, func(t *testing.T) {
			got := IsOpenShiftNamespace(tt.namespace)
			if tt.want != got {
				t.Error(got)
			}
		})
	}
}
