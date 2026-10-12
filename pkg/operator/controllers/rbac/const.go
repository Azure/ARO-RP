package rbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"embed"
)

//go:embed staticresources/*.yaml
var staticResources embed.FS
