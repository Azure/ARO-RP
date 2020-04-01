package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	utilkeyvault "github.com/Azure/ARO-RP/pkg/util/keyvault"
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
	return d.globaldeployments.CreateOrUpdateAndWait(ctx, d.config.Configuration.GlobalResourceGroupName, deploymentName, mgmtresources.Deployment{
		Properties: &mgmtresources.DeploymentProperties{
			Template:   template,
			Mode:       mgmtresources.Incremental,
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
	err = d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, deploymentName, mgmtresources.Deployment{
		Properties: &mgmtresources.DeploymentProperties{
			Template: template,
			Mode:     mgmtresources.Incremental,
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
	return d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, deploymentName, mgmtresources.Deployment{
		Properties: &mgmtresources.DeploymentProperties{
			Template:   template,
			Mode:       mgmtresources.Incremental,
			Parameters: parameters.Parameters,
		},
	})
}

func (d *deployer) configureServiceKV(ctx context.Context) error {
	serviceKeyVaultURI := "https://" + d.config.Configuration.KeyvaultPrefix + "-svc.vault.azure.net/"

	err := d.ensureEncryptionSecret(ctx, serviceKeyVaultURI)
	if err != nil {
		return err
	}

	err = d.ensureMonitoringCertificates(ctx, serviceKeyVaultURI)
	if err != nil {
		return err
	}

	return d.ensureServiceCertificates(ctx, serviceKeyVaultURI)
}

func (d *deployer) ensureEncryptionSecret(ctx context.Context, serviceKeyVaultURI string) error {
	secrets, err := d.keyvault.GetSecrets(ctx, serviceKeyVaultURI, nil)
	if err != nil {
		return err
	}

	for _, secret := range secrets {
		if filepath.Base(*secret.ID) == env.EncryptionSecretName {
			return nil
		}
	}

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		return err
	}

	d.log.Infof("setting %s", env.EncryptionSecretName)
	_, err = d.keyvault.SetSecret(ctx, serviceKeyVaultURI, env.EncryptionSecretName, keyvault.SecretSetParameters{
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

func (d *deployer) ensureServiceCertificates(ctx context.Context, serviceKeyVaultURI string) error {
	_, err := d.keyvault.SetCertificateIssuer(ctx, serviceKeyVaultURI, "OneCert", keyvault.CertificateIssuerSetParameters{
		Provider: to.StringPtr("OneCert"),
	})
	if err != nil {
		return err
	}

	certs := []struct {
		certificateName string
		commonName      string
		eku             utilkeyvault.Eku
		created         bool
	}{
		{
			certificateName: env.RPFirstPartySecretName,
			commonName:      d.config.Configuration.FPServerCertCommonName,
			eku:             utilkeyvault.EkuClientAuth,
		},
		{
			certificateName: env.RPServerSecretName,
			commonName:      "rp." + d.config.Location + "." + d.config.Configuration.RPParentDomainName,
			eku:             utilkeyvault.EkuServerAuth,
		},
	}

	keyVaultCerts, err := d.keyvault.GetCertificates(ctx, serviceKeyVaultURI, nil, nil)
	if err != nil {
		return err
	}

cert:
	for i, c := range certs {
		for _, kc := range keyVaultCerts.Values() {
			// sample id https://aro-int-eastus-svc.vault.azure.net/certificates/rp-server
			u, err := url.Parse(*kc.ID)
			if err != nil {
				return err
			}

			if u.Path == "/certificates/"+c.certificateName && *kc.Attributes.Enabled {
				continue cert
			}
		}

		d.log.Infof("creating %s", c.certificateName)
		err = d.keyvault.CreateSignedCertificate(ctx, serviceKeyVaultURI, utilkeyvault.IssuerOnecert, c.certificateName, c.commonName, c.eku)
		if err != nil {
			return err
		}

		certs[i].created = true
	}

	for _, c := range certs {
		if !c.created {
			continue
		}

		err = d.keyvault.WaitForCertificateOperation(ctx, serviceKeyVaultURI, c.certificateName)
		if err != nil {
			return err
		}
	}

	return nil
}
