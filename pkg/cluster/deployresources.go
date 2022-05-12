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
)

func (m *manager) deployResourceTemplate(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	pg, err := m.graph.LoadPersisted(ctx, resourceGroup, account)
	if err != nil {
		return err
	}

	var installConfig *installconfig.InstallConfig
	var machineMaster *machine.Master
	err = pg.Get(&installConfig, &machineMaster)
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
			m.networkBootstrapNIC(installConfig),
			m.networkMasterNICs(installConfig),
			m.computeBootstrapVM(installConfig),
			m.computeMasterVMs(installConfig, zones, machineMaster),
		},
	}
	return arm.DeployTemplate(ctx, m.log, m.deployments, resourceGroup, "resources", t, map[string]interface{}{
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
