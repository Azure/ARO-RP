package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func ExampleOpenShiftVersionDocument() *OpenShiftVersionDocument {
	return &OpenShiftVersionDocument{
		MissingFields: MissingFields{},
		ID:            "00000000-0000-0000-0000-000000000000",
		OpenShiftVersion: &OpenShiftVersion{
			ID:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resourceGroupName/providers/resourceProviderNamespace/resourceType/resourceName",
			Name: "default",
			Type: "Microsoft.RedHatOpenShift/OpenShiftVersion",
			Properties: OpenShiftVersionProperties{
				Version:           "4.10.20",
				OpenShiftPullspec: "ab:c",
				InstallerPullspec: "de:f",
				Enabled:           true,
			},
		},
	}
}
