package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestAPICertName(t *testing.T) {
	tests := []struct {
		Name                   string
		m                      *manager
		ExpectedCertficateName string
	}{
		{
			Name: "return certificate name",
			m: &manager{
				doc: &api.OpenShiftClusterDocument{
					ID: "f1558837-dd00-479e-a870-37a0e0d2cd6c",
				},
			},
			ExpectedCertficateName: "f1558837-dd00-479e-a870-37a0e0d2cd6c-apiserver",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			actualCertificateName := test.m.APICertName()

			if actualCertificateName != test.ExpectedCertficateName {
				t.Errorf("%s(): expected certificate \"%s\" got \"%s\"\n", test.Name, test.ExpectedCertficateName, actualCertificateName)
			}
		})
	}
}

func TestIngressCertName(t *testing.T) {
	tests := []struct {
		Name                   string
		m                      *manager
		ExpectedCertficateName string
	}{
		{
			Name: "return certificate name",
			m: &manager{
				doc: &api.OpenShiftClusterDocument{
					ID: "f1558837-dd00-479e-a870-37a0e0d2cd6c",
				},
			},
			ExpectedCertficateName: "f1558837-dd00-479e-a870-37a0e0d2cd6c-ingress",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			actualCertificateName := test.m.IngressCertName()

			if actualCertificateName != test.ExpectedCertficateName {
				t.Errorf("%s(): expected certificate \"%s\" got \"%s\"\n", test.Name, test.ExpectedCertficateName, actualCertificateName)
			}
		})
	}
}
