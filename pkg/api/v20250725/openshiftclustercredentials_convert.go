package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
)

type openShiftClusterCredentialsConverter struct{}

// OpenShiftClusterCredentialsToExternal returns a new external representation
// of the internal object, reading from the subset of the internal object's
// fields that appear in the external representation.  ToExternal does not
// modify its argument; there is no pointer aliasing between the passed and
// returned objects.
func (openShiftClusterCredentialsConverter) ToExternal(oc *api.OpenShiftCluster) interface{} {
	out := &OpenShiftClusterCredentials{
		OpenShiftClusterCredentials: generated.OpenShiftClusterCredentials{
			KubeadminUsername: pointerutils.ToPtr("kubeadmin"),
		},
	}
	if oc.Properties.KubeadminPassword != "" {
		out.KubeadminPassword = pointerutils.ToPtr(string(oc.Properties.KubeadminPassword))
	}

	return out
}
