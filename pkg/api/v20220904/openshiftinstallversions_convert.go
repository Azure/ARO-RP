package v20220904

import "github.com/Azure/ARO-RP/pkg/api"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type installOpenShiftVersionsConverter struct{}

// TODO: Change the comment
// OpenShiftClusterCredentialsToExternal returns a new external representation
// of the internal object, reading from the subset of the internal object's
// fields that appear in the external representation.  ToExternal does not
// modify its argument; there is no pointer aliasing between the passed and
// returned objects.
func (*installOpenShiftVersionsConverter) ToExternal(iov *api.InstallOpenShiftVersions) interface{} {
	return iov
}
