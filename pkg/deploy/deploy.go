package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	azcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	azresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/resources"
	"github.com/Azure/ARO-RP/pkg/util/ready"
)

var _ Deployer = (*deployer)(nil)

type Deployer interface {
	PreDeploy(context.Context, *logrus.Entry) (string, error)
	Deploy(context.Context, *logrus.Entry, string) error
	Upgrade(context.Context, *logrus.Entry) error
}

type deployer struct {
	log         *logrus.Entry
	deployments resources.DeploymentsClient
	groups      resources.GroupsClient
	vmss        compute.VirtualMachineScaleSetsClient
	vmssvms     compute.VirtualMachineScaleSetVMsClient
	network     network.PublicIPAddressesClient

	cli *http.Client

	parameters        *arm.Parameters
	version           string
	resourceGroupName string
	subscriptionID    string
	location          string
}

// New initiates new deploy utility object
func New(ctx context.Context, log *logrus.Entry, authorizer autorest.Authorizer, gitCommit string) (*deployer, error) {
	d := &deployer{
		log:               log,
		version:           gitCommit,
		resourceGroupName: os.Getenv("AZURE_RP_RESOURCEGROUP_NAME"),
		subscriptionID:    os.Getenv("AZURE_SUBSCRIPTION_ID"),
		location:          os.Getenv("LOCATION"),
	}

	d.cli = &http.Client{
		Timeout: 5 * time.Second,
	}

	d.deployments = resources.NewDeploymentsClient(d.subscriptionID, authorizer)
	d.groups = resources.NewGroupsClient(d.subscriptionID, authorizer)
	d.vmss = compute.NewVirtualMachineScaleSetsClient(d.subscriptionID, authorizer)
	d.vmssvms = compute.NewVirtualMachineScaleSetVMsClient(d.subscriptionID, authorizer)
	d.network = network.NewPublicIPAddressesClient(d.subscriptionID, authorizer)

	return d, nil
}

// PreDeploy deploys NSG and ManagedIdentity, needed for man deployment
func (d *deployer) PreDeploy(ctx context.Context, log *logrus.Entry) (rpServicePrincipalID string, err error) {
	group := azresources.Group{
		Location: to.StringPtr(d.location),
	}
	_, err = d.groups.CreateOrUpdate(ctx, d.resourceGroupName, group)
	if err != nil {
		return "", err
	}

	deploymentName := "rp-production-nsg"
	var deployment azresources.DeploymentExtended
	deployment, err = d.deployments.Get(ctx, d.resourceGroupName, deploymentName)
	if err != nil {
		if isDeploymentNotFoundError(err) {
			var data []byte
			data, err = Asset(generator.FileRPProductionNSG)
			if err != nil {
				return "", err
			}

			var azuretemplate map[string]interface{}
			err = json.Unmarshal(data, &azuretemplate)
			if err != nil {
				return "", err
			}

			log.Infof("deploying nsg and managedIdentity to %s", d.resourceGroupName)
			err = d.deployments.CreateOrUpdateAndWait(ctx, d.resourceGroupName, deploymentName, azresources.Deployment{
				Properties: &azresources.DeploymentProperties{
					Template: azuretemplate,
					Mode:     azresources.Incremental,
				},
			})
			if err != nil {
				return "", err
			}

			deployment, err = d.deployments.Get(ctx, d.resourceGroupName, "rp-production-nsg")
			if err != nil {
				return
			}
		}
	}
	return deployment.Properties.Outputs.(map[string]interface{})["rpServicePrincipalId"].(map[string]interface{})["value"].(string), nil
}

func (d *deployer) Deploy(ctx context.Context, log *logrus.Entry, rpServicePrincipalID string) error {
	data, err := Asset(generator.FileRPProduction)
	if err != nil {
		return err
	}

	var azuretemplate map[string]interface{}
	err = json.Unmarshal(data, &azuretemplate)
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
	// azure enforce ^[a-z][a-z0-9-]{1,61}[a-z0-9]$. for DomainNameLabel.
	// gitCommit do not comply as it might start with a number
	parameters.Parameters["vmssDomainNameLabel"] = &arm.ParametersParameter{
		Value: "rp-vmss-" + d.version,
	}
	d.parameters = parameters

	rawParameters, err := d.parameters.GetParametersMapInterface()
	if err != nil {
		return err
	}

	log.Infof("deploying rp version %s to %s", d.version, d.resourceGroupName)
	return d.deployments.CreateOrUpdateAndWait(ctx, d.resourceGroupName, "rp-production-"+d.version, azresources.Deployment{
		Properties: &azresources.DeploymentProperties{
			Template:   azuretemplate,
			Mode:       azresources.Incremental,
			Parameters: rawParameters,
		},
	})
}

func (d *deployer) Upgrade(ctx context.Context, log *logrus.Entry) error {
	scaleSets, err := d.vmss.List(ctx, d.resourceGroupName)
	if err != nil {
		return err
	}

	log.Infof("checking new %s RP health", d.version)
	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	err = d.checkRPReadiness(timeoutCtx, log, "rp-vmss-"+d.version)
	if err != nil {
		return err
	}

	log.Info("retire old RP")
	return d.retireOldVMSS(ctx, log, scaleSets)
}

func (d *deployer) checkRPReadiness(ctx context.Context, log *logrus.Entry, vmssName string) error {
	scaleSetsVMs, err := d.vmssvms.List(ctx, d.resourceGroupName, vmssName, "", "", "")
	if err != nil {
		return err
	}

	// construct readiness tracking map with FQDN for checking
	var urlPool []string
	for _, vm := range scaleSetsVMs {
		// note: vmssName matches vmssDomainNameLabel, so we can use it to construct URL
		url := fmt.Sprintf("https://vm%s.%s.%s.cloudapp.azure.com/healthz/ready", *vm.InstanceID, vmssName, d.location)
		urlPool = append(urlPool, url)
	}

	return ready.URLPoolState(ctx, log, d.cli, urlPool, true)
}

func (d *deployer) retireOldVMSS(ctx context.Context, log *logrus.Entry, scaleSets []azcompute.VirtualMachineScaleSet) error {
	for _, vmss := range scaleSets {
		if *vmss.Name == "rp-vmss-"+d.version {
			continue
		}

		log.Info("stop VMSS " + *vmss.Name)
		scaleSetsVMs, err := d.vmssvms.List(ctx, d.resourceGroupName, *vmss.Name, "", "", "")
		if err != nil {
			return err
		}

		// execute individual VMs stop command
		for _, vm := range scaleSetsVMs {
			log.Info("stopping VMS " + *vm.Name)
			err := d.vmssvms.RunCommandAndWait(ctx, d.resourceGroupName, *vmss.Name, *vm.InstanceID, azcompute.RunCommandInput{
				CommandID: to.StringPtr("RunShellScript"),
				Script:    &[]string{"systemctl stop arorp --no-block"},
			})
			if err != nil {
				return err
			}
		}

		var urlPool []string
		for _, vm := range scaleSetsVMs {
			url := fmt.Sprintf("https://vm%s.%s.%s.cloudapp.azure.com/healthz", *vm.InstanceID, *vmss.Name, d.location)
			urlPool = append(urlPool, url)
		}
		err = ready.URLPoolState(ctx, log, d.cli, urlPool, false)
		if err != nil {
			return err
		}

		log.Info("Delete " + *vmss.Name)
		err = d.vmss.DeleteAndWait(ctx, d.resourceGroupName, *vmss.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func getParameters() (*arm.Parameters, error) {
	params, err := ioutil.ReadFile(os.Getenv("AZURE_RP_PARAMETERS_FILE"))
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
