package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func (d *deployer) DeployGateway(ctx context.Context) error {
	// TODO: there is a lot of duplication with DeployRP()
	rpMSI, err := d.userassignedidentities.Get(ctx, d.config.RPResourceGroupName, "aro-rp-"+d.config.Location)
	if err != nil {
		return err
	}

	gwMSI, err := d.userassignedidentities.Get(ctx, d.config.GatewayResourceGroupName, "aro-gateway-"+d.config.Location)
	if err != nil {
		return err
	}

	deploymentName := "gateway-production-" + d.version

	b, err := Asset(generator.FileGatewayProduction)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return err
	}

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))
	parameters.Parameters["dbtokenClientId"] = &arm.ParametersParameter{
		Value: &d.config.Configuration.DBTokenClientID,
	}
	parameters.Parameters["dbtokenURL"] = &arm.ParametersParameter{
		Value: "https://dbtoken." + d.config.Location + "." + *d.config.Configuration.RPParentDomainName + ":8445",
	}
	parameters.Parameters["rpImage"] = &arm.ParametersParameter{
		Value: *d.config.Configuration.RPImagePrefix + ":" + d.version,
	}
	parameters.Parameters["rpResourceGroupName"] = &arm.ParametersParameter{
		Value: d.config.RPResourceGroupName,
	}
	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: rpMSI.PrincipalID.String(),
	}
	parameters.Parameters["gatewayServicePrincipalId"] = &arm.ParametersParameter{
		Value: gwMSI.PrincipalID.String(),
	}
	parameters.Parameters["vmssName"] = &arm.ParametersParameter{
		Value: d.version,
	}

	d.log.Printf("deploying %s", deploymentName)
	return d.deployments.CreateOrUpdateAndWait(ctx, d.config.GatewayResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Mode:       mgmtfeatures.Incremental,
			Parameters: parameters.Parameters,
		},
	})
}
