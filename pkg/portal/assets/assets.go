package assets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"embed"
)

//go:embed v1/*
//go:embed v2/*
var EmbeddedFiles embed.FS
