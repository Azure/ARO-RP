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
				},
				MasterProfile: MasterProfile{
					EncryptionAtHost: EncryptionAtHostDisabled,
				},
				WorkerProfiles: []WorkerProfile{
					{
						EncryptionAtHost: EncryptionAtHostDisabled,
					},
				},
				ClusterProfile: ClusterProfile{
					FipsValidatedModules: FipsValidatedModulesDisabled,
				},
				OperatorFlags: DefaultOperatorFlags(),
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
			name: "default SDN",
			want: func() *OpenShiftClusterDocument {
				return validOpenShiftClusterDocument()
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.NetworkProfile.SoftwareDefinedNetwork = ""
			},
		},
		{
			name: "preserve SDN",
			want: func() *OpenShiftClusterDocument {
				doc := validOpenShiftClusterDocument()
				doc.OpenShiftCluster.Properties.NetworkProfile.SoftwareDefinedNetwork = SoftwareDefinedNetworkOVNKubernetes
				return doc
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.NetworkProfile.SoftwareDefinedNetwork = SoftwareDefinedNetworkOVNKubernetes
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			doc := validOpenShiftClusterDocument()
			want := tt.want()
			if tt.input != nil {
				tt.input(doc)
			}

			SetDefaults(doc)

			if !reflect.DeepEqual(&doc, &want) {
				t.Error(fmt.Errorf("\n%+v\n !=\n%+v", doc, want)) // can't use cmp due to cycle imports
			}
		})
	}
}
