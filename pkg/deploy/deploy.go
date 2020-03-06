package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"time"

	azcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	azresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/resources"
)

var _ Deployer = (*deployer)(nil)

type Deployer interface {
	PreDeploy(context.Context) (string, error)
	Deploy(context.Context, string) error
	Upgrade(context.Context) error
}

type deployer struct {
	log *logrus.Entry

	deployments resources.DeploymentsClient
	groups      resources.GroupsClient
	vmss        compute.VirtualMachineScaleSetsClient
	vmssvms     compute.VirtualMachineScaleSetVMsClient
	network     network.PublicIPAddressesClient

	cli *http.Client

	config  *RPConfig
	version string
}

// New initiates new deploy utility object
func New(ctx context.Context, log *logrus.Entry, authorizer autorest.Authorizer, config *RPConfig, version string) Deployer {
	return &deployer{
		log: log,

		deployments: resources.NewDeploymentsClient(config.SubscriptionID, authorizer),
		groups:      resources.NewGroupsClient(config.SubscriptionID, authorizer),
		vmss:        compute.NewVirtualMachineScaleSetsClient(config.SubscriptionID, authorizer),
		vmssvms:     compute.NewVirtualMachineScaleSetVMsClient(config.SubscriptionID, authorizer),
		network:     network.NewPublicIPAddressesClient(config.SubscriptionID, authorizer),

		cli: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},

		config:  config,
		version: version,
	}
}

// PreDeploy deploys managed identity, NSGs and keyvaults, needed for main
// deployment
func (d *deployer) PreDeploy(ctx context.Context) (string, error) {
	_, err := d.groups.CreateOrUpdate(ctx, d.config.ResourceGroupName, azresources.Group{
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

func (d *deployer) _deployManageIdentity(ctx context.Context, deploymentName string) (azresources.DeploymentExtended, error) {
	b, err := Asset(generator.FileRPProductionManagedIdentity)
	if err != nil {
		return azresources.DeploymentExtended{}, nil
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return azresources.DeploymentExtended{}, nil
	}

	d.log.Infof("deploying managed identity to %s", d.config.ResourceGroupName)
	err = d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, deploymentName, azresources.Deployment{
		Properties: &azresources.DeploymentProperties{
			Template: template,
			Mode:     azresources.Incremental,
		},
	})
	if err != nil {
		return azresources.DeploymentExtended{}, nil
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

	parameters, err := d.getParameters(template)
	if err != nil {
		return err
	}

	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: rpServicePrincipalID,
	}
	d.log.Infof("predeploying to %s", d.config.ResourceGroupName)

	return d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, deploymentName, azresources.Deployment{
		Properties: &azresources.DeploymentProperties{
			Template:   template,
			Mode:       azresources.Incremental,
			Parameters: parameters.Parameters,
		},
	})

}

func (d *deployer) Deploy(ctx context.Context, rpServicePrincipalID string) error {
	b, err := Asset(generator.FileRPProduction)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return err
	}

	parameters, err := d.getParameters(template)
	if err != nil {
		return err
	}

	parameters.Parameters["vmssName"] = &arm.ParametersParameter{
		Value: d.version,
	}
	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: rpServicePrincipalID,
	}

	d.log.Printf("deploying rp version %s to %s", d.version, d.config.ResourceGroupName)
	return d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, "rp-production-"+d.version, azresources.Deployment{
		Properties: &azresources.DeploymentProperties{
			Template:   template,
			Mode:       azresources.Incremental,
			Parameters: parameters.Parameters,
		},
	})
}

func (d *deployer) Upgrade(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	err := d.waitForRPReadiness(timeoutCtx, "rp-vmss-"+d.version)
	if err != nil {
		return err
	}

	return d.removeOldScalesets(ctx)
}

func (d *deployer) waitForRPReadiness(ctx context.Context, vmssName string) error {
	scalesetVMs, err := d.vmssvms.List(ctx, d.config.ResourceGroupName, vmssName, "", "", "")
	if err != nil {
		return err
	}

	d.log.Printf("waiting for %s instances to be healthy", vmssName)
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		for _, vm := range scalesetVMs {
			u := fmt.Sprintf("https://vm%s.%s.%s.cloudapp.azure.com/healthz/ready", *vm.InstanceID, vmssName, d.config.Location)
			resp, err := d.cli.Get(u)
			if err != nil || resp.StatusCode != http.StatusOK {
				d.log.Printf("instance %s not ready", *vm.InstanceID)
				return false, nil
			}
		}

		return true, nil
	}, ctx.Done())
}

func (d *deployer) removeOldScalesets(ctx context.Context) error {
	d.log.Print("removing old scalesets")
	scalesets, err := d.vmss.List(ctx, d.config.ResourceGroupName)
	if err != nil {
		return err
	}

	for _, vmss := range scalesets {
		if *vmss.Name == "rp-vmss-"+d.version {
			continue
		}

		d.log.Printf("stopping scaleset %s", *vmss.Name)
		scalesetVMs, err := d.vmssvms.List(ctx, d.config.ResourceGroupName, *vmss.Name, "", "", "")
		if err != nil {
			return err
		}

		// execute individual VMs stop command
		for _, vm := range scalesetVMs {
			d.log.Printf("stopping instance %s", *vm.Name)
			err := d.vmssvms.RunCommandAndWait(ctx, d.config.ResourceGroupName, *vmss.Name, *vm.InstanceID, azcompute.RunCommandInput{
				CommandID: to.StringPtr("RunShellScript"),
				Script:    &[]string{"systemctl stop arorp --no-block"},
			})
			if err != nil {
				return err
			}
		}

		d.log.Printf("waiting for %s instances to terminate", *vmss.Name)
		err = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
			for _, vm := range scalesetVMs {
				u := fmt.Sprintf("https://vm%s.%s.%s.cloudapp.azure.com/healthz/ready", *vm.InstanceID, *vmss.Name, d.config.Location)

				_, err := d.cli.Get(u)
				if err, ok := err.(*url.Error); !ok || !err.Timeout() {
					d.log.Printf("instance %s not terminated", *vm.InstanceID)
					return false, nil
				}
			}

			return true, nil
		}, ctx.Done())
		if err != nil {
			return err
		}

		d.log.Printf("deleting scaleset %s" + *vmss.Name)
		err = d.vmss.DeleteAndWait(ctx, d.config.ResourceGroupName, *vmss.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) getParameters(ps map[string]interface{}) (*arm.Parameters, error) {
	parameters := &arm.Parameters{
		Parameters: map[string]*arm.ParametersParameter{},
	}
	v := reflect.ValueOf(d.config.Configuration)
	if len(ps) > 0 {
		for paramName := range ps["parameters"].(map[string]interface{}) {
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).Tag.Get("json") == paramName {
					parameters.Parameters[paramName] = &arm.ParametersParameter{
						Value: v.Field(i).Interface(),
					}
				}
			}
		}
	}
	return parameters, nil
}
