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
					SDNProvider: SDNProviderOpenShiftSDN,
				},
				MasterProfile: MasterProfile{
					EncryptionAtHost: EncryptionAtHostDisabled,
				},
				WorkerProfiles: []WorkerProfile{
					{
						EncryptionAtHost: EncryptionAtHostDisabled,
					},
				},
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
				base.OpenShiftCluster.Properties.NetworkProfile.SDNProvider = ""
			},
		},
		{
			name: "preserve SDN",
			want: func() *OpenShiftClusterDocument {
				doc := validOpenShiftClusterDocument()
				doc.OpenShiftCluster.Properties.NetworkProfile.SDNProvider = SDNProviderOVNKubernetes
				return doc
			},
			input: func(base *OpenShiftClusterDocument) {
				base.OpenShiftCluster.Properties.NetworkProfile.SDNProvider = SDNProviderOVNKubernetes
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
