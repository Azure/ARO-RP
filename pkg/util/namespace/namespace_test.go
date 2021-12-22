package namespace

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestIsOpenShift(t *testing.T) {
	for _, tt := range []struct {
		namespace string
		want      bool
	}{
		{
			want: true,
		},
		{
			namespace: "openshift-ns",
			want:      true,
		},
		{
			namespace: "openshift",
			want:      true,
		},
		{
			namespace: "kube-ns",
			want:      true,
		},
		{
			namespace: "default",
			want:      true,
		},
		{
			namespace: "customer",
		},
	} {
		t.Run(tt.namespace, func(t *testing.T) {
			got := IsOpenShift(tt.namespace)
			if tt.want != got {
				t.Error(got)
			}
		})
	}
}

func TestIsOpenShiftSystemNamespace(t *testing.T) {
	for _, tt := range []struct {
		namespace string
		want      bool
	}{
		{
			want: true,
		},
		{
			namespace: "openshift-ns",
			want:      true,
		},
		{
			namespace: "openshift",
			want:      true,
		},
		{
			namespace: "kube-ns",
			want:      true,
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
			got := IsOpenShiftSystemNamespace(tt.namespace)
			if tt.want != got {
				t.Error(got)
			}
		})
	}
}
