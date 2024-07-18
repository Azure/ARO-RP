package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
)

type manager struct {
	log *logrus.Entry
	env env.Interface
	doc *api.OpenShiftClusterDocument

	virtualMachines compute.VirtualMachinesClient
}

func NewFailureDiagnostics(log *logrus.Entry, _env env.Interface,
	doc *api.OpenShiftClusterDocument,

	virtualMachines compute.VirtualMachinesClient,

) *manager {
	return &manager{
		log:             log,
		env:             _env,
		doc:             doc,
		virtualMachines: virtualMachines,
	}
}
