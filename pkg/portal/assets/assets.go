package assets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"embed"
)

//go:embed v2/*
//go:embed prometheus-ui/*
var EmbeddedFiles embed.FS
