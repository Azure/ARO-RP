package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/sirupsen/logrus"
	cryptossh "golang.org/x/crypto/ssh"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armmonitor"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
)

type manager struct {
	log *logrus.Entry
	env env.Interface
	doc *api.OpenShiftClusterDocument

	virtualMachines compute.VirtualMachinesClient
	armInterfaces   armnetwork.InterfacesClient
	loadBalancers   armnetwork.LoadBalancersClient
	armMonitor      armmonitor.MetricsClient

	// bootstrapNodeHostKey holds the SSH host key recorded on the first
	// connection to the bootstrap node (TOFU). Subsequent connections verify
	// against this key. It is nil until the first successful handshake.
	bootstrapNodeHostKey cryptossh.PublicKey
}

func NewFailureDiagnostics(log *logrus.Entry, _env env.Interface,
	doc *api.OpenShiftClusterDocument,

	virtualMachines compute.VirtualMachinesClient,
	armInterfaces armnetwork.InterfacesClient,
	loadBalancers armnetwork.LoadBalancersClient,
	armMonitor armmonitor.MetricsClient,
) *manager {
	return &manager{
		log:             log,
		env:             _env,
		doc:             doc,
		virtualMachines: virtualMachines,
		armInterfaces:   armInterfaces,
		loadBalancers:   loadBalancers,
		armMonitor:      armMonitor,
	}
}
