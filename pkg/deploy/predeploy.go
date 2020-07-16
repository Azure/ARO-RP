package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/to"

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
		_, err = d.groups.CreateOrUpdate(ctx, d.config.Configuration.SubscriptionResourceGroupName, mgmtfeatures.ResourceGroup{
			Location: to.StringPtr("centralus"),
		})
		if err != nil {
			return err
		}

		_, err = d.groups.CreateOrUpdate(ctx, d.config.Configuration.GlobalResourceGroupName, mgmtfeatures.ResourceGroup{
			Location: to.StringPtr("centralus"),
		})
		if err != nil {
			return err
		}

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

	template, err := d.generator.RPGlobalTemplate()
	if err != nil {
		return err
	}

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))
	parameters.Parameters["location"] = &arm.ParametersParameter{
		Value: d.config.Location,
	}
	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: rpServicePrincipalID,
	}

	d.log.Infof("deploying %s", deploymentName)
	return d.globaldeployments.CreateOrUpdateAndWait(ctx, d.config.Configuration.GlobalResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Mode:       mgmtfeatures.Incremental,
			Parameters: parameters.Parameters,
		},
	})
}

func (d *deployer) deployGlobalSubscription(ctx context.Context) error {
	deploymentName := "rp-global-subscription-" + d.config.Location

	template, err := d.generator.RPGlobalSubscriptionTemplate()
	if err != nil {
		return err
	}

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))

	d.log.Infof("deploying %s", deploymentName)
	return d.globaldeployments.CreateOrUpdateAtSubscriptionScopeAndWait(ctx, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Mode:       mgmtfeatures.Incremental,
			Parameters: parameters.Parameters,
		},
		Location: to.StringPtr("centralus"),
	})
}

func (d *deployer) deploySubscription(ctx context.Context) error {
	deploymentName := "rp-production-subscription-" + d.config.Location

	template, err := d.generator.RPSubscriptionTemplate()
	if err != nil {
		return err
	}

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))

	d.log.Infof("deploying %s", deploymentName)
	return d.deployments.CreateOrUpdateAndWait(ctx, d.config.Configuration.SubscriptionResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Mode:       mgmtfeatures.Incremental,
			Parameters: parameters.Parameters,
		},
	})
}

func (d *deployer) deployManagedIdentity(ctx context.Context) error {
	deploymentName := "rp-production-managed-identity"

	template, err := d.generator.ManagedIdentityTemplate()
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

	template, err := d.generator.PreDeployTemplate()
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
	serviceKeyVaultURI := "https://" + d.config.Configuration.KeyvaultPrefix + "-svc.vault.azure.net/"
	secrets, err := d.keyvault.GetSecrets(ctx, serviceKeyVaultURI, nil)
	if err != nil {
		return err
	}

	err = d.ensureSecret(ctx, secrets, serviceKeyVaultURI, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	return d.ensureSecret(ctx, secrets, serviceKeyVaultURI, env.FrontendEncryptionSecretName)
}

func (d *deployer) ensureSecret(ctx context.Context, existingSecrets []keyvault.SecretItem, serviceKeyVaultURI, secretName string) error {
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
	_, err = d.keyvault.SetSecret(ctx, serviceKeyVaultURI, secretName, keyvault.SecretSetParameters{
		Value: to.StringPtr(base64.StdEncoding.EncodeToString(key)),
	})
	return err
}
