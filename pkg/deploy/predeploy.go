package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

// PreDeploy deploys managed identity, NSGs and keyvaults, needed for main
// deployment
func (d *deployer) PreDeploy(ctx context.Context) (string, error) {
	// deploy global rbac
	err := d.deployGlobalSubscription(ctx)
	if err != nil {
		return "", err
	}

	_, err = d.groups.CreateOrUpdate(ctx, d.config.ResourceGroupName, mgmtresources.Group{
		Location: &d.config.Location,
	})
	if err != nil {
		return "", err
	}

	// deploy managed identity if needed and get rpServicePrincipalID
	rpServicePrincipalID, err := d.deployManageIdentity(ctx)
	if err != nil {
		return "", err
	}

	// deploy NSGs, keyvaults
	err = d.deployPreDeploy(ctx, rpServicePrincipalID)
	if err != nil {
		return "", err
	}

	return rpServicePrincipalID, nil
}

func (d *deployer) deployGlobalSubscription(ctx context.Context) error {
	deploymentName := "rp-global-subscription"

	b, err := Asset(generator.FileRPProductionGlobalSubscription)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return err
	}

	d.log.Infof("deploying rbac")
	return d.globaldeployments.CreateOrUpdateAtSubscriptionScopeAndWait(ctx, deploymentName, mgmtresources.Deployment{
		Properties: &mgmtresources.DeploymentProperties{
			Template: template,
			Mode:     mgmtresources.Incremental,
		},
		Location: to.StringPtr("centralus"),
	})
}

func (d *deployer) deployManageIdentity(ctx context.Context) (string, error) {
	deploymentName := "rp-production-managed-identity"

	deployment, err := d.deployments.Get(ctx, d.config.ResourceGroupName, deploymentName)
	if isDeploymentNotFoundError(err) {
		deployment, err = d._deployManageIdentity(ctx, deploymentName)
	}
	if err != nil {
		return "", err
	}

	return deployment.Properties.Outputs.(map[string]interface{})["rpServicePrincipalId"].(map[string]interface{})["value"].(string), nil
}

func (d *deployer) _deployManageIdentity(ctx context.Context, deploymentName string) (mgmtresources.DeploymentExtended, error) {
	b, err := Asset(generator.FileRPProductionManagedIdentity)
	if err != nil {
		return mgmtresources.DeploymentExtended{}, nil
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return mgmtresources.DeploymentExtended{}, nil
	}

	d.log.Infof("deploying managed identity to %s", d.config.ResourceGroupName)
	err = d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, deploymentName, mgmtresources.Deployment{
		Properties: &mgmtresources.DeploymentProperties{
			Template: template,
			Mode:     mgmtresources.Incremental,
		},
	})
	if err != nil {
		return mgmtresources.DeploymentExtended{}, nil
	}

	return d.deployments.Get(ctx, d.config.ResourceGroupName, deploymentName)
}

func (d *deployer) deployPreDeploy(ctx context.Context, rpServicePrincipalID string) error {
	deploymentName := "rp-production-predeploy"

	_, err := d.deployments.Get(ctx, d.config.ResourceGroupName, deploymentName)
	if err == nil || !isDeploymentNotFoundError(err) {
		return err
	}

	b, err := Asset(generator.FileRPProductionPredeploy)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return err
	}

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))
	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: rpServicePrincipalID,
	}

	d.log.Infof("predeploying to %s", d.config.ResourceGroupName)
	return d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, deploymentName, mgmtresources.Deployment{
		Properties: &mgmtresources.DeploymentProperties{
			Template:   template,
			Mode:       mgmtresources.Incremental,
			Parameters: parameters.Parameters,
		},
	})
}
