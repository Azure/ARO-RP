package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/davecgh/go-spew/spew"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

// PreDeploy deploys managed identity, NSGs and keyvaults, needed for main
// deployment
func (d *deployer) PreDeploy(ctx context.Context) error {
	// deploy global rbac
	err := d.deployGlobalSubscription(ctx)
	if err != nil {
		return err
	}

	if d.fullDeploy {
		d.log.Infof("deploying rg %s in %s", *d.config.Configuration.SubscriptionResourceGroupName, *d.config.Configuration.SubscriptionResourceGroupLocation)
		_, err = d.groups.CreateOrUpdate(ctx, *d.config.Configuration.SubscriptionResourceGroupName, mgmtfeatures.ResourceGroup{
			Location: d.config.Configuration.SubscriptionResourceGroupLocation,
		})
		if err != nil {
			return err
		}

		d.log.Infof("deploying rg %s in %s", *d.config.Configuration.GlobalResourceGroupName, *d.config.Configuration.GlobalResourceGroupLocation)
		_, err = d.globalgroups.CreateOrUpdate(ctx, *d.config.Configuration.GlobalResourceGroupName, mgmtfeatures.ResourceGroup{
			Location: d.config.Configuration.GlobalResourceGroupLocation,
		})
		if err != nil {
			return err
		}

		d.log.Infof("deploying rg %s in %s", d.config.ResourceGroupName, d.config.Location)
		_, err = d.groups.CreateOrUpdate(ctx, d.config.ResourceGroupName, mgmtfeatures.ResourceGroup{
			Location: &d.config.Location,
		})
		if err != nil {
			return err
		}
	}

	err = d.deploySubscription(ctx)
	if err != nil {
		return err
	}

	err = d.deployManagedIdentity(ctx)
	if err != nil {
		return err
	}

	msi, err := d.userassignedidentities.Get(ctx, d.config.ResourceGroupName, "aro-rp-"+d.config.Location)
	if err != nil {
		return err
	}

	// Due to https://github.com/Azure/azure-resource-manager-schemas/issues/1067
	// we can't use conditions to define ACR replication object deployment.
	// We use ACRReplicaDisabled in the same way as we use fullDeploy.
	if d.config.Configuration.ACRReplicaDisabled != nil && !*d.config.Configuration.ACRReplicaDisabled {
		err = d.deployGloalACRReplication(ctx)
		if err != nil {
			return err
		}
	}

	err = d.deployGlobal(ctx, msi.PrincipalID.String())
	if err != nil {
		return err
	}

	// deploy NSGs, keyvaults
	err = d.deployPreDeploy(ctx, msi.PrincipalID.String())
	if err != nil {
		return err
	}

	if d.fullDeploy {
		err = d.configureServiceSecrets(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) deployGlobal(ctx context.Context, rpServicePrincipalID string) error {
	deploymentName := "rp-global-" + d.config.Location

	b, err := Asset(generator.FileRPProductionGlobal)
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

	d.log.Infof("deploying %s", deploymentName)
	return d.globaldeployments.CreateOrUpdateAndWait(ctx, *d.config.Configuration.GlobalResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Mode:       mgmtfeatures.Incremental,
			Parameters: parameters.Parameters,
		},
	})
}

func (d *deployer) deployGloalACRReplication(ctx context.Context) error {
	deploymentName := "rp-global-acr-replication-" + d.config.Location

	b, err := Asset(generator.FileRPProductionGlobalACRReplication)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return err
	}

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))
	parameters.Parameters["location"] = &arm.ParametersParameter{
		Value: d.config.Location,
	}

	d.log.Infof("deploying %s", deploymentName)
	return d.globaldeployments.CreateOrUpdateAndWait(ctx, *d.config.Configuration.GlobalResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Mode:       mgmtfeatures.Incremental,
			Parameters: parameters.Parameters,
		},
	})
}

func (d *deployer) deployGlobalSubscription(ctx context.Context) error {
	deploymentName := "rp-global-subscription-" + d.config.Location

	b, err := Asset(generator.FileRPProductionGlobalSubscription)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return err
	}

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))

	d.log.Infof("deploying %s", deploymentName)
	for i := 0; i < 2; i++ {
		err = d.globaldeployments.CreateOrUpdateAtSubscriptionScopeAndWait(ctx, deploymentName, mgmtfeatures.Deployment{
			Properties: &mgmtfeatures.DeploymentProperties{
				Template:   template,
				Mode:       mgmtfeatures.Incremental,
				Parameters: parameters.Parameters,
			},
			Location: to.StringPtr("centralus"),
		})
		if serviceErr, ok := err.(*azure.ServiceError); ok &&
			serviceErr.Code == "RoleDefinitionUpdateConflict" &&
			i == 0 {
			d.log.Print(err)
			continue
		}
		if err != nil {
			// TODO: Remove this once error structure is clear in case of failure
			// and we can "unwrap" it properly for retries
			spew.Dump(err)
			return err
		}

		break
	}
	return nil
}

func (d *deployer) deploySubscription(ctx context.Context) error {
	deploymentName := "rp-production-subscription-" + d.config.Location

	b, err := Asset(generator.FileRPProductionSubscription)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return err
	}

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))

	d.log.Infof("deploying %s", deploymentName)
	return d.deployments.CreateOrUpdateAndWait(ctx, *d.config.Configuration.SubscriptionResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Mode:       mgmtfeatures.Incremental,
			Parameters: parameters.Parameters,
		},
	})
}

func (d *deployer) deployManagedIdentity(ctx context.Context) error {
	deploymentName := "rp-production-managed-identity"

	b, err := Asset(generator.FileRPProductionManagedIdentity)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return err
	}

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))

	d.log.Infof("deploying %s", deploymentName)
	return d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Mode:       mgmtfeatures.Incremental,
			Parameters: parameters.Parameters,
		},
	})
}

func (d *deployer) deployPreDeploy(ctx context.Context, rpServicePrincipalID string) error {
	deploymentName := "rp-production-predeploy"

	var isCreate bool
	_, err := d.deployments.Get(ctx, d.config.ResourceGroupName, deploymentName)
	if isDeploymentNotFoundError(err) {
		isCreate = true
		err = nil
	}
	if err != nil {
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
	parameters.Parameters["deployNSGs"] = &arm.ParametersParameter{
		Value: isCreate,
	}
	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: rpServicePrincipalID,
	}

	d.log.Infof("deploying %s", deploymentName)
	return d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Mode:       mgmtfeatures.Incremental,
			Parameters: parameters.Parameters,
		},
	})
}

func (d *deployer) configureServiceSecrets(ctx context.Context) error {
	secrets, err := d.keyvault.GetSecrets(ctx)
	if err != nil {
		return err
	}

	err = d.ensureSecret(ctx, secrets, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	return d.ensureSecret(ctx, secrets, env.FrontendEncryptionSecretName)
}

func (d *deployer) ensureSecret(ctx context.Context, existingSecrets []keyvault.SecretItem, secretName string) error {
	for _, secret := range existingSecrets {
		if filepath.Base(*secret.ID) == secretName {
			return nil
		}
	}

	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return err
	}

	d.log.Infof("setting %s", secretName)
	return d.keyvault.SetSecret(ctx, secretName, keyvault.SecretSetParameters{
		Value: to.StringPtr(base64.StdEncoding.EncodeToString(key)),
	})
}
