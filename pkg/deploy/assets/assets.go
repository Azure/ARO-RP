package assets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"embed"
)

//go:embed *.json
var EmbeddedFiles embed.FS
