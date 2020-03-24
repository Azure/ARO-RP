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
	"strings"
	"time"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/containerregistry"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/dns"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/resources"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

var _ Deployer = (*deployer)(nil)

type Deployer interface {
	PreDeploy(context.Context) (string, error)
	Deploy(context.Context, string) error
	Upgrade(context.Context) error
}

type deployer struct {
	log *logrus.Entry

	globaldeployments  resources.DeploymentsClient
	globalrecordsets   dns.RecordSetsClient
	globalreplications containerregistry.ReplicationsClient
	deployments        resources.DeploymentsClient
	groups             resources.GroupsClient
	vmss               compute.VirtualMachineScaleSetsClient
	vmssvms            compute.VirtualMachineScaleSetVMsClient
	keyvault           keyvault.Manager

	cli *http.Client

	config  *RPConfig
	version string
}

// New initiates new deploy utility object
func New(ctx context.Context, log *logrus.Entry, config *RPConfig, version string) (Deployer, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	kvAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(azure.PublicCloud.ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, err
	}

	return &deployer{
		log: log,

		globaldeployments:  resources.NewDeploymentsClient(config.Configuration.GlobalSubscriptionID, authorizer),
		globalrecordsets:   dns.NewRecordSetsClient(config.Configuration.GlobalSubscriptionID, authorizer),
		globalreplications: containerregistry.NewReplicationsClient(config.Configuration.GlobalSubscriptionID, authorizer),
		deployments:        resources.NewDeploymentsClient(config.SubscriptionID, authorizer),
		groups:             resources.NewGroupsClient(config.SubscriptionID, authorizer),
		vmss:               compute.NewVirtualMachineScaleSetsClient(config.SubscriptionID, authorizer),
		vmssvms:            compute.NewVirtualMachineScaleSetVMsClient(config.SubscriptionID, authorizer),
		keyvault:           keyvault.NewManager(kvAuthorizer),

		cli: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},

		config:  config,
		version: version,
	}, nil
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

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))
	parameters.Parameters["domainName"] = &arm.ParametersParameter{
		Value: d.config.Location + "." + d.config.Configuration.ClusterParentDomainName,
	}
	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: rpServicePrincipalID,
	}
	parameters.Parameters["vmssName"] = &arm.ParametersParameter{
		Value: d.version,
	}

	d.log.Printf("deploying rp version %s to %s", d.version, d.config.ResourceGroupName)
	err = d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, "rp-production-"+d.version, mgmtresources.Deployment{
		Properties: &mgmtresources.DeploymentProperties{
			Template:   template,
			Mode:       mgmtresources.Incremental,
			Parameters: parameters.Parameters,
		},
	})
	if err != nil {
		return err
	}

	deployment, err := d.deployments.Get(ctx, d.config.ResourceGroupName, "rp-production-"+d.version)
	if err != nil {
		return err
	}

	rpPipIPAddress := deployment.Properties.Outputs.(map[string]interface{})["rp-pip-ipAddress"].(map[string]interface{})["value"].(string)

	_nameServers := deployment.Properties.Outputs.(map[string]interface{})["rp-nameServers"].(map[string]interface{})["value"].([]interface{})
	nameServers := make([]string, 0, len(_nameServers))
	for _, ns := range _nameServers {
		nameServers = append(nameServers, ns.(string))
	}

	err = d.configureDNS(ctx, rpPipIPAddress, nameServers)
	if err != nil {
		return err
	}

	return nil
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
			err := d.vmssvms.RunCommandAndWait(ctx, d.config.ResourceGroupName, *vmss.Name, *vm.InstanceID, mgmtcompute.RunCommandInput{
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

func (d *deployer) configureDNS(ctx context.Context, rpPipIPAddress string, nameServers []string) error {
	_, err := d.globalrecordsets.CreateOrUpdate(ctx, d.config.Configuration.GlobalResourceGroupName, d.config.Configuration.RPParentDomainName, d.config.Location, mgmtdns.A, mgmtdns.RecordSet{
		RecordSetProperties: &mgmtdns.RecordSetProperties{
			TTL: to.Int64Ptr(3600),
			ARecords: &[]mgmtdns.ARecord{
				{
					Ipv4Address: &rpPipIPAddress,
				},
			},
		},
	}, "", "")
	if err != nil {
		return err
	}

	nsRecords := make([]mgmtdns.NsRecord, 0, len(nameServers))
	for i := range nameServers {
		nsRecords = append(nsRecords, mgmtdns.NsRecord{
			Nsdname: &nameServers[i],
		})
	}

	_, err = d.globalrecordsets.CreateOrUpdate(ctx, d.config.Configuration.GlobalResourceGroupName, d.config.Configuration.ClusterParentDomainName, d.config.Location, mgmtdns.NS, mgmtdns.RecordSet{
		RecordSetProperties: &mgmtdns.RecordSetProperties{
			TTL:       to.Int64Ptr(3600),
			NsRecords: &nsRecords,
		},
	}, "", "")
	return err
}

// getParameters returns an *arm.Parameters populated with parameter names and
// values.  The names are taken from the ps argument and the values are taken
// from d.config.Configuration.
func (d *deployer) getParameters(ps map[string]interface{}) *arm.Parameters {
	m := map[string]interface{}{}

	v := reflect.ValueOf(*d.config.Configuration)
	for i := 0; i < v.NumField(); i++ {
		m[strings.SplitN(v.Type().Field(i).Tag.Get("json"), ",", 2)[0]] = v.Field(i).Interface()
	}

	parameters := &arm.Parameters{
		Parameters: map[string]*arm.ParametersParameter{},
	}

	for p := range ps {
		parameters.Parameters[p] = &arm.ParametersParameter{
			Value: m[p],
		}
	}

	return parameters
}
