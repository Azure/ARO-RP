package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OperationList represents an RP operation list.
type OperationList struct {
	// List of operations supported by the resource provider.
	Operations []Operation `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}

// Operation represents an RP operation.
type Operation struct {
	// Operation name: {provider}/{resource}/{operation}.
	Name string `json:"name,omitempty"`

	// The object that describes the operation.
	Display Display `json:"display,omitempty"`

	// Sources of requests to this operation.  Comma separated list with valid values user or system, e.g. "user,system".
	Origin string `json:"origin,omitempty"`
}

// Display represents the display details of an operation.
type Display struct {
	// Friendly name of the resource provider.
	Provider string `json:"provider,omitempty"`

	// Resource type on which the operation is performed.
	Resource string `json:"resource,omitempty"`

	// Operation type: read, write, delete, listKeys/action, etc.
	Operation string `json:"operation,omitempty"`

	// Friendly name of the operation.
	Description string `json:"description,omitempty"`
}

// Common operations defined which can be used within the registration of the APIs
var OperationResultsRead = Operation{
	Name: "Microsoft.RedHatOpenShift/locations/operationresults/read",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "locations/operationresults",
		Operation: "Read operation results",
	},
	Origin: "user,system",
}

var OperationStatusRead = Operation{
	Name: "Microsoft.RedHatOpenShift/locations/operationsstatus/read",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "locations/operationsstatus",
		Operation: "Read operations status",
	},
	Origin: "user,system",
}

var OperationRead = Operation{
	Name: "Microsoft.RedHatOpenShift/operations/read",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "operations",
		Operation: "Read operations",
	},
	Origin: "user,system",
}

var OperationOpenShiftClusterRead = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/read",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "openShiftClusters",
		Operation: "Read OpenShift cluster",
	},
	Origin: "user,system",
}

var OperationOpenShiftClusterWrite = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/write",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "openShiftClusters",
		Operation: "Write OpenShift cluster",
	},
	Origin: "user,system",
}

var OperationOpenShiftClusterDelete = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/delete",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "openShiftClusters",
		Operation: "Delete OpenShift cluster",
	},
	Origin: "user,system",
}

var OperationOpenShiftClusterListCredentials = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/listCredentials/action",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "openShiftClusters",
		Operation: "List credentials of an OpenShift cluster",
	},
	Origin: "user,system",
}

var OperationOpenShiftClusterListAdminCredentials = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/listAdminCredentials/action",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "openShiftClusters",
		Operation: "List Admin Kubeconfig of an OpenShift cluster",
	},
	Origin: "user,system",
}

var OperationOpenShiftClusterGetDetectors = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/detectors/read",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "openShiftClusters",
		Operation: "Get OpenShift Cluster Detector",
	},
	Origin: "user,system",
}

var OperationListInstallVersions = Operation{
	Name: "Microsoft.RedHatOpenShift/locations/listInstallVersions/read",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "listInstallVersions",
		Operation: "Lists all OpenShift versions available to install in the specified location",
	},
	Origin: "user,system",
}

var OperationSyncSetsRead = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/syncSets/read",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "syncSets",
		Operation: "Read OpenShift cluster sync set",
	},
	Origin: "user,system",
}

var OperationSyncSetsWrite = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/syncSets/write",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "syncSets",
		Operation: "Write OpenShift cluster sync set",
	},
	Origin: "user,system",
}

var OperationSyncSetsDelete = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/syncSets/delete",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "syncSets",
		Operation: "Delete OpenShift cluster sync set",
	},
	Origin: "user,system",
}

var OperationMachinePoolsRead = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/machinePools/read",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "machinePools",
		Operation: "Read OpenShift cluster machine pool",
	},
	Origin: "user,system",
}
var OperationMachinePoolsWrite = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/machinePools/write",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "machinePools",
		Operation: "Write OpenShift cluster machine pool",
	},
	Origin: "user,system",
}
var OperationMachinePoolsDelete = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/machinePools/delete",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "machinePools",
		Operation: "Delete OpenShift cluster machine pool",
	},
	Origin: "user,system",
}

var OperationSyncIdentityProvidersRead = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/syncIdentityProviders/read",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "syncIdentityProviders",
		Operation: "Read OpenShift cluster sync identity provider",
	},
	Origin: "user,system",
}
var OperationSyncIdentityProvidersWrite = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/syncIdentityProviders/write",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "syncIdentityProviders",
		Operation: "Write OpenShift cluster sync identity provider",
	},
	Origin: "user,system",
}

var OperationSyncIdentityProvidersDelete = Operation{
	Name: "Microsoft.RedHatOpenShift/openShiftClusters/syncIdentityProviders/delete",
	Display: Display{
		Provider:  "Azure Red Hat OpenShift",
		Resource:  "syncIdentityProviders",
		Operation: "Delete OpenShift cluster sync identity provider",
	},
	Origin: "user,system",
}
