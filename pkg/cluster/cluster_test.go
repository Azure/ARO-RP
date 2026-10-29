package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestStartedInstallerBackend(t *testing.T) {
	tests := []struct {
		Name     string
		doc      *api.OpenShiftClusterDocument
		Expected api.InstallerBackend
	}{
		{
			Name:     "no installation started",
			doc:      &api.OpenShiftClusterDocument{OpenShiftCluster: &api.OpenShiftCluster{}},
			Expected: "",
		},
		{
			Name: "persisted installer profile wins over the rollout flag",
			doc: &api.OpenShiftClusterDocument{OpenShiftCluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					InstallerProfile: &api.InstallerProfile{Backend: api.InstallerBackendAKSJob},
				},
			}},
			Expected: api.InstallerBackendAKSJob,
		},
		{
			Name: "hive namespace implies hive started the install",
			doc: &api.OpenShiftClusterDocument{OpenShiftCluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					HiveProfile: api.HiveProfile{Namespace: "aro-cluster-uuid"},
				},
			}},
			Expected: api.InstallerBackendHive,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			if got := startedInstallerBackend(test.doc); got != test.Expected {
				t.Errorf("got %q, want %q", got, test.Expected)
			}
		})
	}
}

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
