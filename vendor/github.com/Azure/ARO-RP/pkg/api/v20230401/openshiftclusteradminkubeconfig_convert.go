package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type openShiftClusterAdminKubeconfigConverter struct{}

// openShiftClusterAdminKubeconfigConverter returns a new external representation
// of the internal object, reading from the subset of the internal object's
// fields that appear in the external representation.  ToExternal does not
// modify its argument; there is no pointer aliasing between the passed and
// returned objects.
func (openShiftClusterAdminKubeconfigConverter) ToExternal(oc *api.OpenShiftCluster) interface{} {
	return &OpenShiftClusterAdminKubeconfig{
		Kubeconfig: oc.Properties.UserAdminKubeconfig,
	}
}
