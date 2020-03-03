package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
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

	version        string
	resourceGroup  string
	subscriptionID string
	location       string
}

// New initiates new deploy utility object
func New(ctx context.Context, log *logrus.Entry, authorizer autorest.Authorizer, version string) Deployer {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")

	return &deployer{
		log: log,

		deployments: resources.NewDeploymentsClient(subscriptionID, authorizer),
		groups:      resources.NewGroupsClient(subscriptionID, authorizer),
		vmss:        compute.NewVirtualMachineScaleSetsClient(subscriptionID, authorizer),
		vmssvms:     compute.NewVirtualMachineScaleSetVMsClient(subscriptionID, authorizer),
		network:     network.NewPublicIPAddressesClient(subscriptionID, authorizer),

		cli: &http.Client{
			Timeout: 5 * time.Second,
		},

		version:        version,
		resourceGroup:  os.Getenv("RESOURCEGROUP"),
		subscriptionID: subscriptionID,
		location:       os.Getenv("LOCATION"),
	}
}

// PreDeploy deploys NSG and ManagedIdentity, needed for man deployment
func (d *deployer) PreDeploy(ctx context.Context) (rpServicePrincipalID string, err error) {
	group := azresources.Group{
		Location: &d.location,
	}

	_, err = d.groups.CreateOrUpdate(ctx, d.resourceGroup, group)
	if err != nil {
		return "", err
	}

	deploymentName := "rp-production-nsg"
	deployment, err := d.deployments.Get(ctx, d.resourceGroup, deploymentName)
	if isDeploymentNotFoundError(err) {
		var b []byte // must not shadow err
		b, err = Asset(generator.FileRPProductionNSG)
		if err != nil {
			return "", err
		}

		var template map[string]interface{}
		err = json.Unmarshal(b, &template)
		if err != nil {
			return "", err
		}

		d.log.Printf("predeploying to %s", d.resourceGroup)
		err = d.deployments.CreateOrUpdateAndWait(ctx, d.resourceGroup, deploymentName, azresources.Deployment{
			Properties: &azresources.DeploymentProperties{
				Template: template,
				Mode:     azresources.Incremental,
			},
		})
		if err != nil {
			return "", err
		}

		deployment, err = d.deployments.Get(ctx, d.resourceGroup, deploymentName)
	}
	if err != nil {
		return "", err
	}

	return deployment.Properties.Outputs.(map[string]interface{})["rpServicePrincipalId"].(map[string]interface{})["value"].(string), nil
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

	parameters, err := getParameters()
	if err != nil {
		return err
	}

	parameters.Parameters["vmssName"] = &arm.ParametersParameter{
		Value: d.version,
	}
	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: rpServicePrincipalID,
	}

	d.log.Printf("deploying rp version %s to %s", d.version, d.resourceGroup)
	return d.deployments.CreateOrUpdateAndWait(ctx, d.resourceGroup, "rp-production-"+d.version, azresources.Deployment{
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
	scalesetVMs, err := d.vmssvms.List(ctx, d.resourceGroup, vmssName, "", "", "")
	if err != nil {
		return err
	}

	d.log.Printf("waiting for %s instances to be healthy", vmssName)
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		for _, vm := range scalesetVMs {
			u := fmt.Sprintf("https://vm%s.%s.%s.cloudapp.azure.com/healthz/ready", *vm.InstanceID, vmssName, d.location)

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
	scalesets, err := d.vmss.List(ctx, d.resourceGroup)
	if err != nil {
		return err
	}

	for _, vmss := range scalesets {
		if *vmss.Name == "rp-vmss-"+d.version {
			continue
		}

		d.log.Printf("stopping scaleset %s", *vmss.Name)
		scalesetVMs, err := d.vmssvms.List(ctx, d.resourceGroup, *vmss.Name, "", "", "")
		if err != nil {
			return err
		}

		// execute individual VMs stop command
		for _, vm := range scalesetVMs {
			d.log.Printf("stopping instance %s", *vm.Name)
			err := d.vmssvms.RunCommandAndWait(ctx, d.resourceGroup, *vmss.Name, *vm.InstanceID, azcompute.RunCommandInput{
				CommandID: to.StringPtr("RunShellScript"),
				Script:    &[]string{"systemctl stop arorp --no-block"},
			})
			if err != nil {
				return err
			}
		}

		d.log.Printf("waiting for %s instances to terminate", *vmss.Name)
		err = wait.PollImmediateUntil(10*time.Second, func() (ready bool, err error) {
			for _, vm := range scalesetVMs {
				u := fmt.Sprintf("https://vm%s.%s.%s.cloudapp.azure.com/healthz/ready", *vm.InstanceID, *vmss.Name, d.location)

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
		err = d.vmss.DeleteAndWait(ctx, d.resourceGroup, *vmss.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func getParameters() (*arm.Parameters, error) {
	params, err := ioutil.ReadFile(os.Getenv("RP_PARAMETERS_FILE"))
	if err != nil {
		return nil, err
	}

	parameters := &arm.Parameters{}
	err = json.Unmarshal(params, parameters)
	if err != nil {
		return nil, err
	}

	return parameters, nil
}
