package deployment

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"
	"strings"
)

type Mode int

const (
	Production Mode = iota
	Integration
	Development
)

func NewMode() Mode {
	switch strings.ToLower(os.Getenv("RP_MODE")) {
	case "development":
		return Development
	case "int":
		return Integration
	default:
		return Production
	}
}
