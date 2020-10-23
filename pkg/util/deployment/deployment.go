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

	strDevelopment = "development"
	strIntegration = "int"
	strProduction  = "production"
)

func NewMode() Mode {
	switch strings.ToLower(os.Getenv("RP_MODE")) {
	case strDevelopment:
		return Development
	case strIntegration:
		return Integration
	default:
		return Production
	}
}

func (m Mode) String() string {
	switch m {
	case Development:
		return strDevelopment
	case Integration:
		return strIntegration
	default:
		return strProduction
	}
}
