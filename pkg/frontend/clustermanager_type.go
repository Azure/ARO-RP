package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
)

func (f *frontend) readOcmResourceType(vars map[string]string) error {
	badRequestError := fmt.Errorf("the resource type '%s' could not be found in the namespace '%s' for api version '%s'", vars["ocmResourceType"], vars["resourceProviderNamespace"], vars["api-version"])

	switch vars["ocmResourceType"] {
	case "syncset":
		if f.apis[vars["api-version"]].SyncSetConverter == nil {
			return fmt.Errorf("the resource type '%s' could not be found in the namespace '%s' for api version '%s'", vars["ocmResourceType"], vars["resourceProviderNamespace"], vars["api-version"])
		}
	case "machinepool":
		if f.apis[vars["api-version"]].MachinePoolConverter == nil {
			return badRequestError
		}
	case "syncidentityprovider":
		if f.apis[vars["api-version"]].SyncIdentityProviderConverter == nil {
			return badRequestError
		}
	case "secret":
		if f.apis[vars["api-version"]].SecretConverter == nil {
			return badRequestError
		}
	default:
		return fmt.Errorf("the resource type '%s' is not supported for api version '%s'", vars["ocmResourceType"], vars["api-version"])
	}

	return nil
}
