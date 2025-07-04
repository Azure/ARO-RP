package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azmetrics"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
)

type manager struct {
	log *logrus.Entry
	env env.Interface
	doc *api.OpenShiftClusterDocument

	virtualMachines compute.VirtualMachinesClient
	loadBalancers   network.LoadBalancersClient
	metrics         azmetrics.MetricsClient
}

func NewFailureDiagnostics(log *logrus.Entry, _env env.Interface,
	doc *api.OpenShiftClusterDocument,

	virtualMachines compute.VirtualMachinesClient,
	loadBalancers network.LoadBalancersClient,
	metrics azmetrics.MetricsClient,

) *manager {
	return &manager{
		log:             log,
		env:             _env,
		doc:             doc,
		virtualMachines: virtualMachines,
		loadBalancers:   loadBalancers,
		metrics:         metrics,
	}
}
