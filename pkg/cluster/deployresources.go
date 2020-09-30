package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/openshift/installer/pkg/asset/ignition/machine"
	"github.com/openshift/installer/pkg/asset/installconfig"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (m *manager) deployResourceTemplate(ctx context.Context) error {
	g, err := m.loadGraph(ctx)
	if err != nil {
		return err
	}

	installConfig := g[reflect.TypeOf(&installconfig.InstallConfig{})].(*installconfig.InstallConfig)
	machineMaster := g[reflect.TypeOf(&machine.Master{})].(*machine.Master)

	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro" // TODO: remove after deploy
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	vnetID, _, err := subnet.Split(m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	zones, err := zones(installConfig)
	if err != nil {
		return err
	}

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Parameters: map[string]*arm.TemplateParameter{
			"sas": {
				Type: "object",
			},
		},
		Resources: []*arm.Resource{
			dnsPrivateZone(installConfig),
			dnsPrivateRecordAPIINT(infraID, installConfig),
			dnsPrivateRecordAPI(infraID, installConfig),
			dnsEtcdSRVRecord(installConfig),
			dnsEtcdRecords(infraID, installConfig),
			dnsVirtualNetworkLink(vnetID, installConfig),
			networkPrivateLinkService(infraID, m.env.SubscriptionID(), m.doc.OpenShiftCluster, installConfig),
			networkPublicIPAddress(infraID, installConfig),
			networkPublicIPAddressOutbound(infraID, installConfig),
			networkAPIServerPublicLoadBalancer(infraID, m.doc.OpenShiftCluster, installConfig),
			networkInternalLoadBalancer(infraID, m.doc.OpenShiftCluster, installConfig),
			networkPublicLoadBalancer(infraID, m.doc.OpenShiftCluster, installConfig),
			networkBootstrapNIC(infraID, m.doc.OpenShiftCluster, installConfig),
			networkMasterNICs(infraID, m.doc.OpenShiftCluster, installConfig),
			computeBoostrapVM(infraID, m.doc.OpenShiftCluster, installConfig),
			computeMasterVMs(infraID, zones, machineMaster, m.doc.OpenShiftCluster, installConfig),
		},
	}
	return m.deployARMTemplate(ctx, resourceGroup, "resources", t, map[string]interface{}{
		"sas": map[string]interface{}{
			"value": map[string]interface{}{
				"signedStart":         m.doc.OpenShiftCluster.Properties.Install.Now.Format(time.RFC3339),
				"signedExpiry":        m.doc.OpenShiftCluster.Properties.Install.Now.Add(24 * time.Hour).Format(time.RFC3339),
				"signedPermission":    "rl",
				"signedResourceTypes": "o",
				"signedServices":      "b",
				"signedProtocol":      "https",
			},
		},
	})
}

// zones configures how master nodes are distributed across availability zones. In regions where the number of zones matches
// the number of nodes, it's one node per zone. In regions where there are no zones, all the nodes are in the same place.
// Anything else (e.g. 2-zone regions) is currently unsupported.
func zones(installConfig *installconfig.InstallConfig) (zones *[]string, err error) {
	zoneCount := len(installConfig.Config.ControlPlane.Platform.Azure.Zones)
	replicas := int(*installConfig.Config.ControlPlane.Replicas)
	if reflect.DeepEqual(installConfig.Config.ControlPlane.Platform.Azure.Zones, []string{""}) {
		// []string{""} indicates that there are no Azure Zones, so "zones" return value will be nil
	} else if zoneCount == replicas {
		zones = &[]string{"[copyIndex(1)]"}
	} else {
		err = fmt.Errorf("cluster creation with %d zone(s) and %d replica(s) is unimplemented", zoneCount, replicas)
	}
	return
}
