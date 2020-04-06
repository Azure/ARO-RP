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
	"github.com/Azure/go-autorest/autorest/to"

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

	_, err = d.groups.CreateOrUpdate(ctx, d.config.ResourceGroupName, mgmtfeatures.Group{
		Location: &d.config.Location,
	})
	if d.fullDeploy { // upgrade does not have permission to create RG

		// deploy per subscription Action Group
		_, err = d.groups.CreateOrUpdate(ctx, d.config.Configuration.ActionGroupSubscriptionResourceGroupName, mgmtresources.Group{
			Location: to.StringPtr("centralus"),
		})
		if err != nil {
			return err
		}

		_, err = d.groups.CreateOrUpdate(ctx, d.config.ResourceGroupName, mgmtresources.Group{
			Location: &d.config.Location,
		})
		if err != nil {
			return err
		}
	}

	err = d.deployActionGroup(ctx)
	if err != nil {
		return err
	}

	// deploy managed identity
	err = d.deployManageIdentity(ctx)
	if err != nil {
		return err
	}

	rpServicePrincipalID, err := d.getRpServicePrincipalID(ctx)
	if err != nil {
		return err
	}

	err = d.deployGlobal(ctx, rpServicePrincipalID)
	if err != nil {
		return err
	}

	// deploy NSGs, keyvaults
	err = d.deployPreDeploy(ctx, rpServicePrincipalID)
	if err != nil {
		return err
	}

	if d.fullDeploy {
		return d.configureServiceKV(ctx)
	}

	return nil
}

func (d *deployer) deployGlobal(ctx context.Context, rpServicePrincipalID string) error {
	deploymentName := globalRPDeploymentName + "-" + d.config.Location

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
	parameters.Parameters["location"] = &arm.ParametersParameter{
		Value: d.config.Location,
	}
	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: rpServicePrincipalID,
	}
	parameters.Parameters["fullDeploy"] = &arm.ParametersParameter{
		Value: d.fullDeploy,
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
	deploymentName := globalRPSubscriptionDeploymentName + "-" + d.config.Location

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
	parameters.Parameters["fullDeploy"] = &arm.ParametersParameter{
		Value: d.fullDeploy,
	}

	d.log.Infof("deploying %s", deploymentName)
	return d.globaldeployments.CreateOrUpdateAtSubscriptionScopeAndWait(ctx, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template: template,
			Mode:     mgmtfeatures.Incremental,
		},
		Location: to.StringPtr("centralus"),
	})
}

func (d *deployer) deploySubscription(ctx context.Context) error {
	deploymentName := rpProductionActionGroupDeploymentName + "-" + d.config.Location

	b, err := Asset(generator.FileRPProductionSubscription)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return err
	}

	d.log.Infof("deploying %s", deploymentName)
	return d.deployments.CreateOrUpdateAndWait(ctx, d.config.Configuration.ActionGroupSubscriptionResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template: template,
			Mode:     mgmtfeatures.Incremental,
		},
	})
}

func (d *deployer) deployManageIdentity(ctx context.Context) error {
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
	parameters.Parameters["fullDeploy"] = &arm.ParametersParameter{
		Value: d.fullDeploy,
	}

	d.log.Infof("deploying %s", rpManagedIdentityDeploymentName)
	return d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, rpManagedIdentityDeploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template: template,
			Mode:     mgmtfeatures.Incremental,
		},
	})
}

func (d *deployer) deployPreDeploy(ctx context.Context, rpServicePrincipalID string) error {
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
	parameters.Parameters["fullDeploy"] = &arm.ParametersParameter{
		Value: d.fullDeploy,
	}
	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: rpServicePrincipalID,
	}

	d.log.Infof("deploying %s", rpPredeploymentName)
	return d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, rpPredeploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Mode:       mgmtfeatures.Incremental,
			Parameters: parameters.Parameters,
		},
	})
}

func (d *deployer) configureServiceKV(ctx context.Context) error {
	serviceKeyVaultURI := "https://" + d.config.Configuration.KeyvaultPrefix + "-svc.vault.azure.net/"
	secrets, err := d.keyvault.GetSecrets(ctx, serviceKeyVaultURI, nil)
	if err != nil {
		return err
	}

	err = d.ensureSecret(ctx, secrets, serviceKeyVaultURI, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	err = d.ensureSecret(ctx, secrets, serviceKeyVaultURI, env.FrontendEncryptionSecretName)
	if err != nil {
		return err
	}

	return d.ensureMonitoringCertificates(ctx, serviceKeyVaultURI)
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

func (d *deployer) ensureMonitoringCertificates(ctx context.Context, serviceKeyVaultURI string) error {
	for _, certificateName := range []string{
		env.ClusterLoggingSecretName,
		env.RPLoggingSecretName,
		env.RPMonitoringSecretName,
	} {
		bundle, err := d.keyvault.GetSecret(ctx, d.config.Configuration.GlobalMonitoringKeyVaultURI, certificateName, "")
		if err != nil {
			return err
		}

		d.log.Infof("importing %s", certificateName)
		_, err = d.keyvault.ImportCertificate(ctx, serviceKeyVaultURI, certificateName, keyvault.CertificateImportParameters{
			Base64EncodedCertificate: bundle.Value,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) getRpServicePrincipalID(ctx context.Context) (string, error) {
	msi, err := d.msi.Get(ctx, d.config.ResourceGroupName, "aro-rp-"+d.config.Location)
	if err != nil {
		return "", err
	}
	return msi.UserAssignedIdentityProperties.PrincipalID.String(), nil
}
