package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"time"

	azkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

// PreDeploy deploys managed identity, NSGs and keyvaults, needed for main
// deployment
func (d *deployer) PreDeploy(ctx context.Context) error {
	err := d.enableEncryptionAtHostSubscriptionFeatureFlag(ctx)
	if err != nil {
		return err
	}

	// deploy global rbac
	err = d.deployRPGlobalSubscription(ctx)
	if err != nil {
		return err
	}

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

	// deploy action groups
	err = d.deployRPSubscription(ctx)
	if err != nil {
		return err
	}

	// deploy managed identity
	err = d.deployRPManagedIdentity(ctx)
	if err != nil {
		return err
	}

	rpMSI, err := d.userassignedidentities.Get(ctx, d.config.ResourceGroupName, "aro-rp-"+d.config.Location)
	if err != nil {
		return err
	}

	// Due to https://github.com/Azure/azure-resource-manager-schemas/issues/1067
	// we can't use conditions to define ACR replication object deployment.
	if d.config.Configuration.ACRReplicaDisabled != nil && !*d.config.Configuration.ACRReplicaDisabled {
		err = d.deployRPGlobalACRReplication(ctx)
		if err != nil {
			return err
		}
	}

	// deploy ACR RBAC, RP version storage account
	err = d.deployRPGlobal(ctx, rpMSI.PrincipalID.String())
	if err != nil {
		return err
	}

	// deploy NSGs, keyvaults
	err = d.deployRPPreDeploy(ctx, rpMSI.PrincipalID.String())
	if err != nil {
		return err
	}

	err = d.configureServiceSecrets(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (d *deployer) enableEncryptionAtHostSubscriptionFeatureFlag(ctx context.Context) error {
	f, err := d.features.Get(ctx, "Microsoft.Compute", "EncryptionAtHost")
	if err != nil {
		return err
	}

	if *f.Properties.State == "Registered" {
		return nil
	}

	d.log.Print("registering Microsoft.Compute/EncryptionAtHost feature")
	_, err = d.features.Register(ctx, "Microsoft.Compute", "EncryptionAtHost")
	if err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	err = wait.PollImmediateUntil(time.Minute, func() (bool, error) {
		f, err = d.features.Get(ctx, "Microsoft.Compute", "EncryptionAtHost")
		if err != nil {
			return false, err
		}

		return *f.Properties.State == "Registered", nil
	}, timeoutCtx.Done())
	if err != nil {
		return err
	}

	_, err = d.providers.Register(ctx, "Microsoft.Compute")
	return err
}

func (d *deployer) deployRPGlobal(ctx context.Context, rpServicePrincipalID string) error {
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

func (d *deployer) deployRPGlobalACRReplication(ctx context.Context) error {
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

func (d *deployer) deployRPGlobalSubscription(ctx context.Context) error {
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

	d.log.Infof("deploying %s", deploymentName)
	for i := 0; i < 5; i++ {
		err = d.globaldeployments.CreateOrUpdateAtSubscriptionScopeAndWait(ctx, deploymentName, mgmtfeatures.Deployment{
			Properties: &mgmtfeatures.DeploymentProperties{
				Template: template,
				Mode:     mgmtfeatures.Incremental,
			},
			Location: to.StringPtr("centralus"),
		})
		if serviceErr, ok := err.(*azure.ServiceError); ok &&
			serviceErr.Code == "DeploymentFailed" &&
			i < 4 {
			// Sometimes we see RoleDefinitionUpdateConflict when multiple RPs
			// are deploying at once.  Retry a few times.
			d.log.Print(err)
			continue
		}
		if err != nil {
			return err
		}

		break
	}
	return nil
}

func (d *deployer) deployRPSubscription(ctx context.Context) error {
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

	d.log.Infof("deploying %s", deploymentName)
	return d.deployments.CreateOrUpdateAndWait(ctx, *d.config.Configuration.SubscriptionResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template: template,
			Mode:     mgmtfeatures.Incremental,
		},
	})
}

func (d *deployer) deployRPManagedIdentity(ctx context.Context) error {
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

	d.log.Infof("deploying %s", deploymentName)
	return d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template: template,
			Mode:     mgmtfeatures.Incremental,
		},
	})
}

func (d *deployer) deployRPPreDeploy(ctx context.Context, rpServicePrincipalID string) error {
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
	err := d.ensureSecret(ctx, d.serviceKeyvault, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	err = d.ensureSecret(ctx, d.serviceKeyvault, env.FrontendEncryptionSecretName)
	if err != nil {
		return err
	}

	err = d.ensureSecret(ctx, d.portalKeyvault, env.PortalServerSessionKeySecretName)
	if err != nil {
		return err
	}

	return d.ensureSecretKey(ctx, d.portalKeyvault, env.PortalServerSSHKeySecretName)
}

func (d *deployer) ensureSecret(ctx context.Context, kv keyvault.Manager, secretName string) error {
	existingSecrets, err := kv.GetSecrets(ctx)
	if err != nil {
		return err
	}

	for _, secret := range existingSecrets {
		if filepath.Base(*secret.ID) == secretName {
			return nil
		}
	}

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		return err
	}

	d.log.Infof("setting %s", secretName)
	return kv.SetSecret(ctx, secretName, azkeyvault.SecretSetParameters{
		Value: to.StringPtr(base64.StdEncoding.EncodeToString(key)),
	})
}

func (d *deployer) ensureSecretKey(ctx context.Context, kv keyvault.Manager, secretName string) error {
	existingSecrets, err := kv.GetSecrets(ctx)
	if err != nil {
		return err
	}

	for _, secret := range existingSecrets {
		if filepath.Base(*secret.ID) == secretName {
			return nil
		}
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	d.log.Infof("setting %s", secretName)
	return kv.SetSecret(ctx, secretName, azkeyvault.SecretSetParameters{
		Value: to.StringPtr(base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(key))),
	})
}
