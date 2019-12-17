package v20191231preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/jim-minter/rp/pkg/api"
)

// openShiftClusterCredentialsToExternal returns a new external representation
// of the internal object, reading from the subset of the internal object's
// fields that appear in the external representation.  ToExternal does not
// modify its argument; there is no pointer aliasing between the passed and
// returned objects.
func openShiftClusterCredentialsToExternal(oc *api.OpenShiftCluster) *OpenShiftClusterCredentials {
	out := &OpenShiftClusterCredentials{
		KubeadminPassword: oc.Properties.KubeadminPassword,
	}

	return out
}
