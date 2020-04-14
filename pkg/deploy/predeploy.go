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
func (d *deployer) PreDeploy(ctx context.Context) (string, error) {
	// deploy global rbac
	err := d.deployGlobalSubscription(ctx)
	if err != nil {
		return "", err
	}

	// deploy per subscription Action Group
	_, err = d.groups.CreateOrUpdate(ctx, d.config.Configuration.SubscriptionResourceGroupName, mgmtfeatures.ResourceGroup{
		Location: to.StringPtr("centralus"),
	})
	if err != nil {
		return "", err
	}

	err = d.deploySubscription(ctx)
	if err != nil {
		return "", err
	}

	_, err = d.groups.CreateOrUpdate(ctx, d.config.ResourceGroupName, mgmtfeatures.ResourceGroup{
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

	err = d.deployGlobal(ctx, rpServicePrincipalID)
	if err != nil {
		return "", err
	}

	// deploy NSGs, keyvaults
	err = d.deployPreDeploy(ctx, rpServicePrincipalID)
	if err != nil {
		return "", err
	}

	err = d.configureServiceKV(ctx)
	if err != nil {
		return "", err
	}

	return rpServicePrincipalID, nil
}

func (d *deployer) deployGlobal(ctx context.Context, rpServicePrincipalID string) error {
	deploymentName := "rp-global"

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

	d.log.Infof("deploying %s", deploymentName)
	return d.globaldeployments.CreateOrUpdateAtSubscriptionScopeAndWait(ctx, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template: template,
			Mode:     mgmtfeatures.Incremental,
		},
		Location: to.StringPtr("centralus"),
	})
}

func (d *deployer) deployActionGroup(ctx context.Context) error {
	deploymentName := "rp-production-subscription"

	b, err := Asset(generator.FileRPProductionActionGroup)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return err
	}

	d.log.Infof("deploying %s", deploymentName)
	return d.deployments.CreateOrUpdateAndWait(ctx, d.config.Configuration.SubscriptionResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template: template,
			Mode:     mgmtfeatures.Incremental,
		},
	})
}

func (d *deployer) deployManageIdentity(ctx context.Context) (string, error) {
	deploymentName := "rp-production-managed-identity"

	b, err := Asset(generator.FileRPProductionManagedIdentity)
	if err != nil {
		return "", err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return "", err
	}

	d.log.Infof("deploying %s", deploymentName)
	err = d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template: template,
			Mode:     mgmtfeatures.Incremental,
		},
	})
	if err != nil {
		return "", err
	}

	deployment, err := d.deployments.Get(ctx, d.config.ResourceGroupName, deploymentName)
	if err != nil {
		return "", err
	}

	return deployment.Properties.Outputs.(map[string]interface{})["rpServicePrincipalId"].(map[string]interface{})["value"].(string), nil
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
