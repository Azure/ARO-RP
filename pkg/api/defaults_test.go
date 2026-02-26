package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"reflect"
	"testing"
)

func validOpenShiftClusterDocument() *OpenShiftClusterDocument {
	doc := OpenShiftClusterDocument{
		OpenShiftCluster: &OpenShiftCluster{
			Properties: OpenShiftClusterProperties{
				NetworkProfile: NetworkProfile{
					SoftwareDefinedNetwork: SoftwareDefinedNetworkOpenShiftSDN,
					OutboundType:           OutboundTypeLoadbalancer,
					PreconfiguredNSG:       PreconfiguredNSGDisabled,
					LoadBalancerProfile: &LoadBalancerProfile{
						ManagedOutboundIPs: &ManagedOutboundIPs{
							Count: 1,
						},
					},
				},
				MasterProfile: MasterProfile{
					EncryptionAtHost: EncryptionAtHostDisabled,
				},
				WorkerProfiles: []WorkerProfile{
					{
						EncryptionAtHost: EncryptionAtHostDisabled,
					},
				},
				WorkerProfilesStatus: []WorkerProfile{
					{
						EncryptionAtHost: EncryptionAtHostDisabled,
					},
				},
				ClusterProfile: ClusterProfile{
					FipsValidatedModules: FipsValidatedModulesDisabled,
				},
				OperatorFlags: OperatorFlags{"testflag": "testvalue"},
			},
		},
	}

	return &doc
}

func TestSetDefaults(t *testing.T) {
	for _, tt := range []struct {
		name  string
		want  func() *OpenShiftClusterDocument
		input func(doc *OpenShiftClusterDocument)
	}{
		{
			name: "no defaults needed",
			want: func() *OpenShiftClusterDocument {
				return validOpenShiftClusterDocument()
			},
		},
		{
			name: "default encryption at host",
			want: func() *OpenShiftClusterDocument {
				return validOpenShiftClusterDocument()
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.MasterProfile.EncryptionAtHost = ""
			},
		},
		{
			name: "preserve encryption at host",
			want: func() *OpenShiftClusterDocument {
				doc := validOpenShiftClusterDocument()
				doc.OpenShiftCluster.Properties.MasterProfile.EncryptionAtHost = EncryptionAtHostEnabled
				return doc
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.MasterProfile.EncryptionAtHost = EncryptionAtHostEnabled
			},
		},
		{
			name: "default fips validated modules",
			want: func() *OpenShiftClusterDocument {
				return validOpenShiftClusterDocument()
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.ClusterProfile.FipsValidatedModules = ""
			},
		},
		{
			name: "preserve fips validated modules",
			want: func() *OpenShiftClusterDocument {
				doc := validOpenShiftClusterDocument()
				doc.OpenShiftCluster.Properties.ClusterProfile.FipsValidatedModules = FipsValidatedModulesEnabled
				return doc
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.ClusterProfile.FipsValidatedModules = FipsValidatedModulesEnabled
			},
		},
		{
			name: "default flags",
			want: func() *OpenShiftClusterDocument {
				return validOpenShiftClusterDocument()
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.OperatorFlags = nil
			},
		},
		{
			name: "preserve flags",
			want: func() *OpenShiftClusterDocument {
				doc := validOpenShiftClusterDocument()
				doc.OpenShiftCluster.Properties.OperatorFlags = OperatorFlags{}
				return doc
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.OperatorFlags = OperatorFlags{}
			},
		},
		{
			name: "default lb profile",
			want: func() *OpenShiftClusterDocument {
				doc := validOpenShiftClusterDocument()
				return doc
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile = nil
			},
		},
		// DNS defaults: auto-detect from version (empty/unset aro.dns.type)
		{
			name: "dns auto-detect - version 4.21 sets clusterhosted",
			want: func() *OpenShiftClusterDocument {
				doc := validOpenShiftClusterDocument()
				doc.OpenShiftCluster.Properties.ClusterProfile.Version = "4.21.0"
				doc.OpenShiftCluster.Properties.OperatorFlags[dnsTypeFlagKey] = dnsTypeClusterHosted
				return doc
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.ClusterProfile.Version = "4.21.0"
			},
		},
		{
			name: "dns auto-detect - version above 4.21 sets clusterhosted",
			want: func() *OpenShiftClusterDocument {
				doc := validOpenShiftClusterDocument()
				doc.OpenShiftCluster.Properties.ClusterProfile.Version = "4.22.1"
				doc.OpenShiftCluster.Properties.OperatorFlags[dnsTypeFlagKey] = dnsTypeClusterHosted
				return doc
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.ClusterProfile.Version = "4.22.1"
			},
		},
		{
			name: "dns auto-detect - version below 4.21 does not set dns type",
			want: func() *OpenShiftClusterDocument {
				doc := validOpenShiftClusterDocument()
				doc.OpenShiftCluster.Properties.ClusterProfile.Version = "4.20.5"
				return doc
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.ClusterProfile.Version = "4.20.5"
			},
		},
		{
			name: "dns auto-detect - empty version does not set dns type",
			want: func() *OpenShiftClusterDocument {
				return validOpenShiftClusterDocument()
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.ClusterProfile.Version = ""
			},
		},
		// DNS defaults: explicit dnsmasq is always accepted
		{
			name: "dns switch - dnsmasq accepted on 4.21+ cluster",
			want: func() *OpenShiftClusterDocument {
				doc := validOpenShiftClusterDocument()
				doc.OpenShiftCluster.Properties.ClusterProfile.Version = "4.21.0"
				doc.OpenShiftCluster.Properties.OperatorFlags[dnsTypeFlagKey] = dnsTypeDnsmasq
				return doc
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.ClusterProfile.Version = "4.21.0"
				base.OpenShiftCluster.Properties.OperatorFlags[dnsTypeFlagKey] = dnsTypeDnsmasq
			},
		},
		{
			name: "dns switch - dnsmasq accepted on pre-4.21 cluster",
			want: func() *OpenShiftClusterDocument {
				doc := validOpenShiftClusterDocument()
				doc.OpenShiftCluster.Properties.ClusterProfile.Version = "4.20.0"
				doc.OpenShiftCluster.Properties.OperatorFlags[dnsTypeFlagKey] = dnsTypeDnsmasq
				return doc
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.ClusterProfile.Version = "4.20.0"
				base.OpenShiftCluster.Properties.OperatorFlags[dnsTypeFlagKey] = dnsTypeDnsmasq
			},
		},
		// DNS defaults: explicit clusterhosted validated against version
		{
			name: "dns switch - clusterhosted accepted on 4.21+ cluster",
			want: func() *OpenShiftClusterDocument {
				doc := validOpenShiftClusterDocument()
				doc.OpenShiftCluster.Properties.ClusterProfile.Version = "4.21.0"
				doc.OpenShiftCluster.Properties.OperatorFlags[dnsTypeFlagKey] = dnsTypeClusterHosted
				return doc
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.ClusterProfile.Version = "4.21.0"
				base.OpenShiftCluster.Properties.OperatorFlags[dnsTypeFlagKey] = dnsTypeClusterHosted
			},
		},
		{
			name: "dns switch - clusterhosted rejected on pre-4.21 cluster, cleared to default",
			want: func() *OpenShiftClusterDocument {
				doc := validOpenShiftClusterDocument()
				doc.OpenShiftCluster.Properties.ClusterProfile.Version = "4.20.0"
				doc.OpenShiftCluster.Properties.OperatorFlags[dnsTypeFlagKey] = ""
				return doc
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.ClusterProfile.Version = "4.20.0"
				base.OpenShiftCluster.Properties.OperatorFlags[dnsTypeFlagKey] = dnsTypeClusterHosted
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			doc := validOpenShiftClusterDocument()
			want := tt.want()
			if tt.input != nil {
				tt.input(doc)
			}

			SetDefaults(doc, func() map[string]string { return map[string]string{"testflag": "testvalue"} })

			if !reflect.DeepEqual(&doc, &want) {
				t.Error(fmt.Errorf("\n%+v\n !=\n%+v", doc, want)) // can't use cmp due to cycle imports
			}
		})
	}
}
