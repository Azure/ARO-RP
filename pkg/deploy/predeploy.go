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
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	azsecretssdk "github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/deploy/assets"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
)

const (
	// Rotate the secret on every deploy of the RP if the most recent
	// secret is greater than 7 days old
	rotateSecretAfter = time.Hour * 24 * 7
	rpRestartScript   = "systemctl restart aro-monitor; systemctl restart aro-portal; systemctl restart aro-rp"
)

// PreDeploy deploys managed identity, NSGs and keyvaults, needed for main
// deployment
func (d *deployer) PreDeploy(ctx context.Context, lbHealthcheckWaitTimeSec int) error {
	// deploy global rbac
	err := d.deployRPGlobalSubscription(ctx)
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

	d.log.Infof("deploying rg %s in %s", d.config.RPResourceGroupName, d.config.Location)
	_, err = d.groups.CreateOrUpdate(ctx, d.config.RPResourceGroupName, mgmtfeatures.ResourceGroup{
		Location: &d.config.Location,
	})
	if err != nil {
		return err
	}

	d.log.Infof("deploying rg %s in %s", d.config.GatewayResourceGroupName, d.config.Location)
	_, err = d.groups.CreateOrUpdate(ctx, d.config.GatewayResourceGroupName, mgmtfeatures.ResourceGroup{
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
	err = d.deployManagedIdentity(ctx, d.config.RPResourceGroupName, generator.FileRPProductionManagedIdentity)
	if err != nil {
		return err
	}

	rpMSI, err := d.userassignedidentities.Get(ctx, d.config.RPResourceGroupName, "aro-rp-"+d.config.Location)
	if err != nil {
		return err
	}

	// deploy managed identity
	err = d.deployManagedIdentity(ctx, d.config.GatewayResourceGroupName, generator.FileGatewayProductionManagedIdentity)
	if err != nil {
		return err
	}

	gwMSI, err := d.userassignedidentities.Get(ctx, d.config.GatewayResourceGroupName, "aro-gateway-"+d.config.Location)
	if err != nil {
		return err
	}

	globalDevopsMSI, err := d.globaluserassignedidentities.Get(ctx, *d.config.Configuration.GlobalResourceGroupName, *d.config.Configuration.GlobalDevopsManagedIdentity)
	if err != nil {
		return err
	}

	// deploy ACR RBAC, RP version storage account
	err = d.deployRPGlobal(ctx, rpMSI.PrincipalID.String(), gwMSI.PrincipalID.String(), globalDevopsMSI.PrincipalID.String())
	if err != nil {
		return err
	}

	// Due to https://github.com/Azure/azure-resource-manager-schemas/issues/1067
	// we can't use conditions to define ACR replication object deployment.
	// Also, an ACR replica cannot be defined in the home registry location.
	acrLocation := *d.config.Configuration.GlobalResourceGroupLocation
	if d.config.Configuration.ACRLocationOverride != nil && *d.config.Configuration.ACRLocationOverride != "" {
		acrLocation = *d.config.Configuration.ACRLocationOverride
	}
	if !strings.EqualFold(d.config.Location, acrLocation) &&
		(d.config.Configuration.ACRReplicaDisabled == nil || !*d.config.Configuration.ACRReplicaDisabled) {
		err = d.deployRPGlobalACRReplication(ctx)
		if err != nil {
			return err
		}
	}

	// deploy NSGs, keyvaults
	// gateway first because RP predeploy will peer its vnet to the gateway vnet

	// key the decision to deploy NSGs on the existence of the gateway
	// predeploy.  We do this in order to refresh the RP NSGs when the gateway
	// is deployed for the first time.
	isCreate := false
	_, err = d.deployments.Get(ctx, d.config.GatewayResourceGroupName, strings.TrimSuffix(generator.FileGatewayProductionPredeploy, ".json"))
	if isDeploymentNotFoundError(err) {
		isCreate = true
		err = nil
	}
	if err != nil {
		return err
	}

	err = d.deployPreDeploy(ctx, d.config.GatewayResourceGroupName, generator.FileGatewayProductionPredeploy, "gatewayServicePrincipalId", gwMSI.PrincipalID.String(), isCreate)
	if err != nil {
		return err
	}

	err = d.deployPreDeploy(ctx, d.config.RPResourceGroupName, generator.FileRPProductionPredeploy, "rpServicePrincipalId", rpMSI.PrincipalID.String(), isCreate)
	if err != nil {
		return err
	}

	return d.configureServiceSecrets(ctx, lbHealthcheckWaitTimeSec)
}

func (d *deployer) deployRPGlobal(ctx context.Context, rpServicePrincipalID, gatewayServicePrincipalID, devopsServicePrincipalId string) error {
	deploymentName := "rp-global-" + d.config.Location

	asset, err := assets.EmbeddedFiles.ReadFile(generator.FileRPProductionGlobal)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(asset, &template)
	if err != nil {
		return err
	}

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))
	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: rpServicePrincipalID,
	}
	parameters.Parameters["gatewayServicePrincipalId"] = &arm.ParametersParameter{
		Value: gatewayServicePrincipalID,
	}
	parameters.Parameters["globalDevopsServicePrincipalId"] = &arm.ParametersParameter{
		Value: devopsServicePrincipalId,
	}

	for i := 0; i < 2; i++ {
		d.log.Infof("deploying %s", deploymentName)
		err = d.globaldeployments.CreateOrUpdateAndWait(ctx, *d.config.Configuration.GlobalResourceGroupName, deploymentName, mgmtfeatures.Deployment{
			Properties: &mgmtfeatures.DeploymentProperties{
				Template:   template,
				Mode:       mgmtfeatures.Incremental,
				Parameters: parameters.Parameters,
			},
		})
		if serviceErr, ok := err.(*azure.ServiceError); ok &&
			serviceErr.Code == "DeploymentFailed" &&
			i < 1 {
			// Can get a Conflict ("Another operation is in progress") on the
			// ACR.  Retry once.
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

func (d *deployer) deployRPGlobalACRReplication(ctx context.Context) error {
	deploymentName := "rp-global-acr-replication-" + d.config.Location

	asset, err := assets.EmbeddedFiles.ReadFile(generator.FileRPProductionGlobalACRReplication)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(asset, &template)
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

	asset, err := assets.EmbeddedFiles.ReadFile(generator.FileRPProductionGlobalSubscription)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(asset, &template)
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
			Location: d.config.Configuration.GlobalResourceGroupLocation,
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

	asset, err := assets.EmbeddedFiles.ReadFile(generator.FileRPProductionSubscription)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(asset, &template)
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

func (d *deployer) deployManagedIdentity(ctx context.Context, resourceGroupName, deploymentFile string) error {
	deploymentName := strings.TrimSuffix(deploymentFile, ".json")

	asset, err := assets.EmbeddedFiles.ReadFile(deploymentFile)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(asset, &template)
	if err != nil {
		return err
	}

	d.log.Infof("deploying %s", deploymentName)
	return d.deployments.CreateOrUpdateAndWait(ctx, resourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template: template,
			Mode:     mgmtfeatures.Incremental,
		},
	})
}

func (d *deployer) deployPreDeploy(ctx context.Context, resourceGroupName, deploymentFile, spIDName, spID string, isCreate bool) error {
	deploymentName := strings.TrimSuffix(deploymentFile, ".json")

	asset, err := assets.EmbeddedFiles.ReadFile(deploymentFile)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(asset, &template)
	if err != nil {
		return err
	}

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))
	parameters.Parameters["deployNSGs"] = &arm.ParametersParameter{
		Value: isCreate,
	}
	// TODO: ugh
	if _, ok := template["parameters"].(map[string]interface{})["gatewayResourceGroupName"]; ok {
		parameters.Parameters["gatewayResourceGroupName"] = &arm.ParametersParameter{
			Value: d.config.GatewayResourceGroupName,
		}
	}
	parameters.Parameters[spIDName] = &arm.ParametersParameter{
		Value: spID,
	}

	d.log.Infof("deploying %s", deploymentName)
	return d.deployments.CreateOrUpdateAndWait(ctx, resourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Mode:       mgmtfeatures.Incremental,
			Parameters: parameters.Parameters,
		},
	})
}

func (d *deployer) configureServiceSecrets(ctx context.Context, lbHealthcheckWaitTimeSec int) error {
	isRotated := false
	for _, s := range []struct {
		kv         azsecrets.Client
		secretName string
		len        int
	}{
		{d.serviceKeyvault, env.EncryptionSecretV2Name, 64},
		{d.serviceKeyvault, env.FrontendEncryptionSecretV2Name, 64},
		{d.portalKeyvault, env.PortalServerSessionKeySecretName, 32},
	} {
		isNew, err := d.ensureAndRotateSecret(ctx, s.kv, s.secretName, s.len)
		isRotated = isNew || isRotated
		if err != nil {
			return err
		}
	}

	// don't rotate legacy secrets
	for _, s := range []struct {
		kv         azsecrets.Client
		secretName string
		len        int
	}{
		{d.serviceKeyvault, env.EncryptionSecretName, 32},
		{d.serviceKeyvault, env.FrontendEncryptionSecretName, 32},
	} {
		isNew, err := d.ensureSecret(ctx, s.kv, s.secretName, s.len)
		isRotated = isNew || isRotated
		if err != nil {
			return err
		}
	}

	isNew, err := d.ensureSecretKey(ctx, d.portalKeyvault, env.PortalServerSSHKeySecretName)
	isRotated = isNew || isRotated
	if err != nil {
		return err
	}

	if isRotated {
		err = d.restartOldScalesets(ctx, lbHealthcheckWaitTimeSec)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *deployer) ensureAndRotateSecret(ctx context.Context, kv azsecrets.Client, secretName string, len int) (isNew bool, err error) {
	pager := kv.NewListSecretPropertiesPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, err
		}
		for _, item := range page.Value {
			if item == nil || item.ID == nil {
				continue
			}
			if filepath.Base((*item.ID).Name()) == secretName {
				latestVersion, err := kv.GetSecret(ctx, secretName, "", nil)
				if err != nil {
					return false, err
				}

				updatedTime := latestVersion.Attributes.Created.Add(rotateSecretAfter)

				// do not create a secret if rotateSecretAfter time has
				// not elapsed since the secret version's creation timestamp
				if time.Now().Before(updatedTime) {
					return false, nil
				}
			}
		}
	}

	return true, d.createSecret(ctx, kv, secretName, len)
}

func (d *deployer) ensureSecret(ctx context.Context, kv azsecrets.Client, secretName string, len int) (isNew bool, err error) {
	pager := kv.NewListSecretPropertiesPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, err
		}
		for _, item := range page.Value {
			if item == nil || item.ID == nil {
				continue
			}
			if filepath.Base((*item.ID).Name()) == secretName {
				return false, nil
			}
		}
	}

	return true, d.createSecret(ctx, kv, secretName, len)
}

func (d *deployer) createSecret(ctx context.Context, kv azsecrets.Client, secretName string, len int) error {
	key := make([]byte, len)
	_, err := rand.Read(key)
	if err != nil {
		return err
	}

	d.log.Infof("setting %s", secretName)
	_, err = kv.SetSecret(ctx, secretName, azsecretssdk.SetSecretParameters{
		Value: to.StringPtr(base64.StdEncoding.EncodeToString(key)),
	}, nil)
	return err
}

func (d *deployer) ensureSecretKey(ctx context.Context, kv azsecrets.Client, secretName string) (isNew bool, err error) {
	pager := kv.NewListSecretPropertiesPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, err
		}
		for _, item := range page.Value {
			if item == nil || item.ID == nil {
				continue
			}
			if filepath.Base((*item.ID).Name()) == secretName {
				return false, nil
			}
		}
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return false, err
	}

	d.log.Infof("setting %s", secretName)
	_, err = kv.SetSecret(ctx, secretName, azsecretssdk.SetSecretParameters{
		Value: to.StringPtr(base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(key))),
	}, nil)
	return true, err
}

func (d *deployer) restartOldScalesets(ctx context.Context, lbHealthcheckWaitTimeSec int) error {
	scalesets, err := d.vmss.List(ctx, d.config.RPResourceGroupName)
	if err != nil {
		return err
	}

	for _, vmss := range scalesets {
		err = d.restartOldScaleset(ctx, *vmss.Name, lbHealthcheckWaitTimeSec)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) restartOldScaleset(ctx context.Context, vmssName string, lbHealthcheckWaitTimeSec int) error {
	if !strings.HasPrefix(vmssName, rpVMSSPrefix) {
		return &api.CloudError{
			StatusCode: http.StatusBadRequest,
			CloudErrorBody: &api.CloudErrorBody{
				Code: api.CloudErrorCodeInvalidResource,
				Message: fmt.Sprintf("provided vmss %s does not match RP prefix",
					vmssName,
				),
			},
		}
	}

	scalesetVMs, err := d.vmssvms.List(ctx, d.config.RPResourceGroupName, vmssName, "", "", "")
	if err != nil {
		return err
	}

	for _, vm := range scalesetVMs {
		d.log.Printf("waiting for restart script to complete on older rp vmss %s, instance %s", vmssName, *vm.InstanceID)
		err = d.vmssvms.RunCommandAndWait(ctx, d.config.RPResourceGroupName, vmssName, *vm.InstanceID, mgmtcompute.RunCommandInput{
			CommandID: to.StringPtr("RunShellScript"),
			Script:    &[]string{rpRestartScript},
		})

		if err != nil {
			return err
		}

		// wait for load balancer probe to change the vm health status
		time.Sleep(time.Duration(lbHealthcheckWaitTimeSec) * time.Second)
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Hour)
		defer cancel()
		err = d.waitForReadiness(timeoutCtx, vmssName, *vm.InstanceID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) waitForReadiness(ctx context.Context, vmssName string, vmInstanceID string) error {
	return wait.PollUntilContextCancel(ctx, 10*time.Second, true, func(ctx context.Context) (bool, error) {
		return d.isVMInstanceHealthy(ctx, d.config.RPResourceGroupName, vmssName, vmInstanceID), nil
	})
}

func (d *deployer) isVMInstanceHealthy(ctx context.Context, resourceGroupName string, vmssName string, vmInstanceID string) bool {
	r, err := d.vmssvms.GetInstanceView(ctx, resourceGroupName, vmssName, vmInstanceID)
	instanceUnhealthy := r.VMHealth != nil && r.VMHealth.Status != nil && r.VMHealth.Status.Code != nil && *r.VMHealth.Status.Code != "HealthState/healthy"
	if err != nil || instanceUnhealthy {
		d.log.Printf("instance %s is unhealthy", vmInstanceID)
		return false
	}
	return true
}
