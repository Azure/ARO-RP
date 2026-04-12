package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type openShiftClusterCredentialsConverter struct{}

// OpenShiftClusterCredentialsToExternal returns a new external representation
// of the internal object, reading from the subset of the internal object's
// fields that appear in the external representation.  ToExternal does not
// modify its argument; there is no pointer aliasing between the passed and
// returned objects.
func (openShiftClusterCredentialsConverter) ToExternal(oc *api.OpenShiftCluster) interface{} {
	out := &OpenShiftClusterCredentials{
		KubeadminUsername: "kubeadmin",
		KubeadminPassword: string(oc.Properties.KubeadminPassword),
	}

	return out
}
