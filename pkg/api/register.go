package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/env"
)

type OpenShiftClusterConverter interface {
	ToExternal(*OpenShiftCluster) interface{}
	ToExternalList([]*OpenShiftCluster) interface{}
	ToInternal(interface{}, *OpenShiftCluster)
}

type OpenShiftClusterValidator interface {
	Static(interface{}, *OpenShiftCluster) error
	Dynamic(context.Context, *OpenShiftCluster) error
}

type OpenShiftClusterCredentialsConverter interface {
	ToExternal(*OpenShiftCluster) interface{}
}

// Version is a set of endpoints implemented by each API version
type Version struct {
	OpenShiftClusterConverter            func() OpenShiftClusterConverter
	OpenShiftClusterValidator            func(env.Interface, string) OpenShiftClusterValidator
	OpenShiftClusterCredentialsConverter func() OpenShiftClusterCredentialsConverter
}

// APIs is the map of registered API versions
var APIs = map[string]*Version{}
