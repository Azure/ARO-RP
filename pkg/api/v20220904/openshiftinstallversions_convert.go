package v20220904

import "github.com/Azure/ARO-RP/pkg/api"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type installOpenShiftVersionsConverter struct{}

func (*installOpenShiftVersionsConverter) ToExternal(iov *api.InstallOpenShiftVersions) interface{} {
	return iov
}
