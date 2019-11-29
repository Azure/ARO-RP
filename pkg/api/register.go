package api

import (
	"context"
)

// External is the interface that an external API must implement
type External interface {
	Validate(context.Context, string, string, *OpenShiftCluster) error
	ToInternal(*OpenShiftCluster)
}

// APIVersionType represents an APIVersion and a Type
type APIVersionType struct {
	APIVersion string
	Type       string
}

// APIs is the map of registered external APIs
var APIs = map[APIVersionType]func(*OpenShiftCluster) External{}
