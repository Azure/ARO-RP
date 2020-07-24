package machine

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	machinev1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
)

func IsMasterRole(m *machinev1beta1.Machine) (bool, error) {
	role, roleOK := m.Labels["machine.openshift.io/cluster-api-machine-role"]
	if !roleOK {
		return false, fmt.Errorf("role not set on machine label yet")
	}
	return role == "master", nil
}
