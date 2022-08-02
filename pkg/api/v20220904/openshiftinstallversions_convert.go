package v20220904

import "github.com/Azure/ARO-RP/pkg/api"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type installVersionsConverter struct{}

func (*installVersionsConverter) ToExternal(iov *api.InstallVersions) interface{} {
	return iov
}
