package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/dns"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/msi"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

var _ Deployer = (*deployer)(nil)

type Deployer interface {
	PreDeploy(context.Context) error
	Deploy(context.Context) error
	Upgrade(context.Context) error
}

type deployer struct {
	log *logrus.Entry

	globaldeployments features.DeploymentsClient
	globalrecordsets  dns.RecordSetsClient
	deployments       features.DeploymentsClient
	dns               dns.ZonesClient
	groups            features.ResourceGroupsClient
	vmss              compute.VirtualMachineScaleSetsClient
	vmssvms           compute.VirtualMachineScaleSetVMsClient
	publicipaddresses network.PublicIPAddressesClient
	msi               msi.UserAssignedIdentitiesClient
	keyvault          keyvault.Manager

	config     *RPConfig
	version    string
	fullDeploy bool
}

// New initiates new deploy utility object
func New(ctx context.Context, log *logrus.Entry, config *RPConfig, version string, fullDeploy bool) (Deployer, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	kvAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(azure.PublicCloud.ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, err
	}

	d := &deployer{
		log: log,

		globaldeployments: features.NewDeploymentsClient(config.Configuration.GlobalSubscriptionID, authorizer),
		globalrecordsets:  dns.NewRecordSetsClient(config.Configuration.GlobalSubscriptionID, authorizer),
		deployments:       features.NewDeploymentsClient(config.SubscriptionID, authorizer),
		dns:               dns.NewZonesClient(config.SubscriptionID, authorizer),
		groups:            features.NewResourceGroupsClient(config.SubscriptionID, authorizer),
		vmss:              compute.NewVirtualMachineScaleSetsClient(config.SubscriptionID, authorizer),
		vmssvms:           compute.NewVirtualMachineScaleSetVMsClient(config.SubscriptionID, authorizer),
		publicipaddresses: network.NewPublicIPAddressesClient(config.SubscriptionID, authorizer),
		msi:               msi.NewUserAssignedIdentitiesClient(config.SubscriptionID, authorizer),
		keyvault:          keyvault.NewManager(kvAuthorizer),

		config:  config,
		version: version,

		fullDeploy: fullDeploy,
	}

	return d, err
}

func (d *deployer) Deploy(ctx context.Context) error {
	deploymentName := "rp-production-" + d.version

	b, err := Asset(generator.FileRPProduction)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return err
	}

	rpServicePrincipalID, err := d.getRpServicePrincipalID(ctx)
	if err != nil {
		return err
	}

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))
	parameters.Parameters["adminApiCaBundle"] = &arm.ParametersParameter{
		Value: base64.StdEncoding.EncodeToString([]byte(d.config.Configuration.AdminAPICABundle)),
	}
	parameters.Parameters["domainName"] = &arm.ParametersParameter{
		Value: d.config.Location + "." + d.config.Configuration.ClusterParentDomainName,
	}
	parameters.Parameters["extraCosmosDBIPs"] = &arm.ParametersParameter{
		Value: strings.Join(d.config.Configuration.ExtraCosmosDBIPs, ","),
	}
	parameters.Parameters["rpImage"] = &arm.ParametersParameter{
		Value: d.config.Configuration.RPImagePrefix + ":" + d.version,
	}
	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: rpServicePrincipalID,
	}
	parameters.Parameters["vmssName"] = &arm.ParametersParameter{
		Value: d.version,
	}
	parameters.Parameters["fullDeploy"] = &arm.ParametersParameter{
		Value: d.fullDeploy,
	}

	d.log.Printf("deploying %s", deploymentName)
	err = d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   template,
			Mode:       mgmtfeatures.Incremental,
			Parameters: parameters.Parameters,
		},
	})
	if err != nil {
		return err
	}

	if d.fullDeploy {
		ip, err := d.publicipaddresses.Get(ctx, d.config.ResourceGroupName, "rp-pip", "")
		if err != nil {
			return err
		}

		dnsZone, err := d.dns.ListByResourceGroup(ctx, d.config.ResourceGroupName, nil)
		if err != nil {
			return err
		}

		var nameServers []string
		dnsZoneName := fmt.Sprintf("%s.%s", d.config.Location, d.config.Configuration.ClusterParentDomainName)
		for _, z := range dnsZone {
			if *z.Name == dnsZoneName {
				nameServers = *z.ZoneProperties.NameServers
			}
		}
		if len(nameServers) == 0 {
			return fmt.Errorf("no nameserver found for dns zone %s", dnsZoneName)
		}

		err = d.configureDNS(ctx, *ip.PublicIPAddressPropertiesFormat.IPAddress, nameServers)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) configureDNS(ctx context.Context, rpPipIPAddress string, nameServers []string) error {
	_, err := d.globalrecordsets.CreateOrUpdate(ctx, d.config.Configuration.GlobalResourceGroupName, d.config.Configuration.RPParentDomainName, "rp."+d.config.Location, mgmtdns.A, mgmtdns.RecordSet{
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
