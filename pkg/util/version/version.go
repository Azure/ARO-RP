package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	apiversion "github.com/Azure/ARO-RP/pkg/api/util/version"
)

// Re-export apiversion variants of this
type Version apiversion.Version

var (
	NewVersion   = apiversion.NewVersion
	ParseVersion = apiversion.ParseVersion
)
