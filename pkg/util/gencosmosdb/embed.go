package gencosmosdb

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"embed"
)

//go:embed cosmosdb/*.go
var EmbeddedFiles embed.FS
