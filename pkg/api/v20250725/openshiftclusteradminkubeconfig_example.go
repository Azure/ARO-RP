package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"

	"github.com/Azure/ARO-RP/pkg/api/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
)

// ExampleOpenShiftClusterAdminKubeconfigResponse returns an example
// OpenShiftClusterAdminKubeconfig object that the RP might return to an end-user
func ExampleOpenShiftClusterAdminKubeconfigResponse() interface{} {
	return &OpenShiftClusterAdminKubeconfig{
		OpenShiftClusterAdminKubeconfig: generated.OpenShiftClusterAdminKubeconfig{
			Kubeconfig: pointerutils.ToPtr(base64.StdEncoding.EncodeToString([]byte("{}"))),
		},
	}
}
