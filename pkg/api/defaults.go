package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// SetDefaults sets the default values for older api version
// when interacting with newer api versions. This together with
// database migration will make sure we have right values in the cluster documents
// when moving between old and new versions
func SetDefaults(doc *OpenShiftClusterDocument) {
	// SDNProvider was introduced in 2021-09-01-preview
	if doc.OpenShiftCluster != nil {
		if doc.OpenShiftCluster.Properties.NetworkProfile.SDNProvider == "" {
			doc.OpenShiftCluster.Properties.NetworkProfile.SDNProvider = SDNProviderOpenShiftSDN
		}

		// EncryptionAtHost was introduced in 2021-09-01-preview.
		// It can't be changed post cluster creation
		if doc.OpenShiftCluster.Properties.MasterProfile.EncryptionAtHost == "" {
			doc.OpenShiftCluster.Properties.MasterProfile.EncryptionAtHost = EncryptionAtHostDisabled
		}

		for i, wp := range doc.OpenShiftCluster.Properties.WorkerProfiles {
			if wp.EncryptionAtHost == "" {
				doc.OpenShiftCluster.Properties.WorkerProfiles[i].EncryptionAtHost = EncryptionAtHostDisabled
			}
		}
	}
}
