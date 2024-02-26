package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// ExampleOperationListResponse returns an example OperationList object that the
// RP might return to an end-user
func ExampleOperationListResponse() interface{} {
	return &OperationList{
		Operations: []Operation{
			{
				Name: "Microsoft.RedHatOpenShift/openShiftClusters/read",
				Display: Display{
					Provider:  "Azure Red Hat OpenShift",
					Resource:  "openShiftClusters",
					Operation: "Read OpenShift cluster",
				},
			},
		},
	}
}
