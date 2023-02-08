package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
)

func (f *frontend) validateOcmResourceType(apiVersion, ocmResourceType string) error {
	badRequestError := fmt.Errorf("the resource type '%s' is not valid for api version '%s'", ocmResourceType, apiVersion)

	switch ocmResourceType {
	case "syncset":
		if f.apis[apiVersion].SyncSetConverter == nil {
			return badRequestError
		}
	case "machinepool":
		if f.apis[apiVersion].MachinePoolConverter == nil {
			return badRequestError
		}
	case "syncidentityprovider":
		if f.apis[apiVersion].SyncIdentityProviderConverter == nil {
			return badRequestError
		}
	case "secret":
		if f.apis[apiVersion].SecretConverter == nil {
			return badRequestError
		}
	default:
		return badRequestError
	}

	return nil
}
