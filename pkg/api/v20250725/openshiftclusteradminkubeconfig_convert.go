package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
)

type openShiftClusterAdminKubeconfigConverter struct{}

// openShiftClusterAdminKubeconfigConverter returns a new external representation
// of the internal object, reading from the subset of the internal object's
// fields that appear in the external representation.  ToExternal does not
// modify its argument; there is no pointer aliasing between the passed and
// returned objects.
func (openShiftClusterAdminKubeconfigConverter) ToExternal(oc *api.OpenShiftCluster) interface{} {
	out := &OpenShiftClusterAdminKubeconfig{
		OpenShiftClusterAdminKubeconfig: generated.OpenShiftClusterAdminKubeconfig{},
	}
	if len(oc.Properties.UserAdminKubeconfig) > 0 {
		out.Kubeconfig = pointerutils.ToPtr(base64.StdEncoding.EncodeToString(oc.Properties.UserAdminKubeconfig))
	}

	return out
}
