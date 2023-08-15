package installer

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
	resourceGroup := stringutils.LastTokenByte(m.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.oc.Properties.StorageSuffix

	pg, err := m.graph.LoadPersisted(ctx, resourceGroup, account)
	if err != nil {
		return err
	}

	var installConfig *installconfig.InstallConfig
	var machineMaster *machine.Master
	err = pg.Get(true, &installConfig, &machineMaster)
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
				"signedStart":         m.oc.Properties.Install.Now.Format(time.RFC3339),
				"signedExpiry":        m.oc.Properties.Install.Now.Add(24 * time.Hour).Format(time.RFC3339),
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
// Valid zone values are nil, 1, 2, and 3. Greater than 3 zones is not supported.
func zones(installConfig *installconfig.InstallConfig) (zones *[]string, err error) {
	zoneCount := len(installConfig.Config.ControlPlane.Platform.Azure.Zones)
	replicas := int(*installConfig.Config.ControlPlane.Replicas)

	if zoneCount > replicas || replicas > 3 {
		err = fmt.Errorf("cluster creation with %d zone(s) and %d replica(s) is unsupported", zoneCount, replicas)
	} else if reflect.DeepEqual(installConfig.Config.ControlPlane.Platform.Azure.Zones, []string{""}) {
		return
	} else if zoneCount <= 2 {
		zones = &installConfig.Config.ControlPlane.Platform.Azure.Zones
	} else {
		zones = &[]string{"[copyIndex(1)]"}
	}

	return
}
