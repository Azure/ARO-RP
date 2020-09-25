package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/insights"
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

	globaldeployments      features.DeploymentsClient
	globalrecordsets       dns.RecordSetsClient
	deployments            features.DeploymentsClient
	groups                 features.ResourceGroupsClient
	metricalerts           insights.MetricAlertsClient
	userassignedidentities msi.UserAssignedIdentitiesClient
	publicipaddresses      network.PublicIPAddressesClient
	vmss                   compute.VirtualMachineScaleSetsClient
	vmssvms                compute.VirtualMachineScaleSetVMsClient
	zones                  dns.ZonesClient
	keyvault               keyvault.Manager

	fullDeploy bool
	config     *RPConfig
	version    string
}

// New initiates new deploy utility object
func New(ctx context.Context, log *logrus.Entry, config *RPConfig, version string, fullDeploy bool) (Deployer, error) {
	err := config.validate()
	if err != nil {
		return nil, err
	}

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

		globaldeployments:      features.NewDeploymentsClient(*config.Configuration.GlobalSubscriptionID, authorizer),
		globalrecordsets:       dns.NewRecordSetsClient(*config.Configuration.GlobalSubscriptionID, authorizer),
		deployments:            features.NewDeploymentsClient(config.SubscriptionID, authorizer),
		groups:                 features.NewResourceGroupsClient(config.SubscriptionID, authorizer),
		metricalerts:           insights.NewMetricAlertsClient(config.SubscriptionID, authorizer),
		userassignedidentities: msi.NewUserAssignedIdentitiesClient(config.SubscriptionID, authorizer),
		publicipaddresses:      network.NewPublicIPAddressesClient(config.SubscriptionID, authorizer),
		vmss:                   compute.NewVirtualMachineScaleSetsClient(config.SubscriptionID, authorizer),
		vmssvms:                compute.NewVirtualMachineScaleSetVMsClient(config.SubscriptionID, authorizer),
		zones:                  dns.NewZonesClient(config.SubscriptionID, authorizer),
		keyvault:               keyvault.NewManager(kvAuthorizer, "https://"+*config.Configuration.KeyvaultPrefix+"-svc.vault.azure.net/"),

		fullDeploy: fullDeploy,
		config:     config,
		version:    version,
	}, nil
}

func (d *deployer) Deploy(ctx context.Context) error {
	msi, err := d.userassignedidentities.Get(ctx, d.config.ResourceGroupName, "aro-rp-"+d.config.Location)
	if err != nil {
		return err
	}

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

	parameters := d.getParameters(template["parameters"].(map[string]interface{}))
	parameters.Parameters["adminApiCaBundle"] = &arm.ParametersParameter{
		Value: base64.StdEncoding.EncodeToString([]byte(*d.config.Configuration.AdminAPICABundle)),
	}
	parameters.Parameters["domainName"] = &arm.ParametersParameter{
		Value: d.config.Location + "." + *d.config.Configuration.ClusterParentDomainName,
	}
	parameters.Parameters["extraCosmosDBIPs"] = &arm.ParametersParameter{
		Value: strings.Join(d.config.Configuration.ExtraCosmosDBIPs, ","),
	}
	parameters.Parameters["rpImage"] = &arm.ParametersParameter{
		Value: *d.config.Configuration.RPImagePrefix + ":" + d.version,
	}
	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: msi.PrincipalID.String(),
	}
	parameters.Parameters["vmssName"] = &arm.ParametersParameter{
		Value: d.version,
	}

	for i := 0; i < 2; i++ {
		d.log.Printf("deploying %s", deploymentName)
		err = d.deployments.CreateOrUpdateAndWait(ctx, d.config.ResourceGroupName, deploymentName, mgmtfeatures.Deployment{
			Properties: &mgmtfeatures.DeploymentProperties{
				Template:   template,
				Mode:       mgmtfeatures.Incremental,
				Parameters: parameters.Parameters,
			},
		})
		if serviceErr, ok := err.(*azure.ServiceError); ok &&
			serviceErr.Code == "DeploymentFailed" &&
			d.fullDeploy &&
			i == 0 {
			// on new RP deployments, we get a spurious DeploymentFailed error
			// from the Microsoft.Insights/metricAlerts resources indicating
			// that rp-lb can't be found, even though it exists and the
			// resources correctly have a dependsOn stanza referring to it.
			// Retry once.
			d.log.Print(err)
			continue
		}
		if err != nil {
			return err
		}

		break
	}

	if d.fullDeploy {
		err = d.configureDNS(ctx)
		if err != nil {
			return err
		}

		err = d.removeOldMetricAlerts(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) configureDNS(ctx context.Context) error {
	rpPip, err := d.publicipaddresses.Get(ctx, d.config.ResourceGroupName, "rp-pip", "")
	if err != nil {
		return err
	}

	zone, err := d.zones.Get(ctx, d.config.ResourceGroupName, d.config.Location+"."+*d.config.Configuration.ClusterParentDomainName)
	if err != nil {
		return err
	}

	_, err = d.globalrecordsets.CreateOrUpdate(ctx, *d.config.Configuration.GlobalResourceGroupName, *d.config.Configuration.RPParentDomainName, "rp."+d.config.Location, mgmtdns.A, mgmtdns.RecordSet{
		RecordSetProperties: &mgmtdns.RecordSetProperties{
			TTL: to.Int64Ptr(3600),
			ARecords: &[]mgmtdns.ARecord{
				{
					Ipv4Address: rpPip.IPAddress,
				},
			},
		},
	}, "", "")
	if err != nil {
		return err
	}

	nsRecords := make([]mgmtdns.NsRecord, 0, len(*zone.NameServers))
	for i := range *zone.NameServers {
		nsRecords = append(nsRecords, mgmtdns.NsRecord{
			Nsdname: &(*zone.NameServers)[i],
		})
	}

	_, err = d.globalrecordsets.CreateOrUpdate(ctx, *d.config.Configuration.GlobalResourceGroupName, *d.config.Configuration.ClusterParentDomainName, d.config.Location, mgmtdns.NS, mgmtdns.RecordSet{
		RecordSetProperties: &mgmtdns.RecordSetProperties{
			TTL:       to.Int64Ptr(3600),
			NsRecords: &nsRecords,
		},
	}, "", "")
	return err
}

// removeOldMetricAlerts removes alert rules without the location in the name
func (d *deployer) removeOldMetricAlerts(ctx context.Context) error {
	d.log.Print("removing old alerts")
	metricAlerts, err := d.metricalerts.ListByResourceGroup(ctx, d.config.ResourceGroupName)
	if err != nil {
		return err
	}

	if metricAlerts.Value == nil {
		return nil
	}

	for _, metricAlert := range *metricAlerts.Value {
		switch *metricAlert.Name {
		case "rp-availability-alert", "rp-degraded-alert", "rp-vnet-alert":
			_, err = d.metricalerts.Delete(ctx, d.config.ResourceGroupName, *metricAlert.Name)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// getParameters returns an *arm.Parameters populated with parameter names and
// values.  The names are taken from the ps argument and the values are taken
// from d.config.Configuration.
func (d *deployer) getParameters(ps map[string]interface{}) *arm.Parameters {
	m := map[string]interface{}{}
	v := reflect.ValueOf(*d.config.Configuration)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsNil() {
			continue
		}

		m[strings.SplitN(v.Type().Field(i).Tag.Get("json"), ",", 2)[0]] = v.Field(i).Interface()
	}

	parameters := &arm.Parameters{
		Parameters: map[string]*arm.ParametersParameter{},
	}

	for p := range ps {
		// do not convert empty fields
		// makes default values templates work
		v, ok := m[p]
		if !ok {
			continue
		}

		parameters.Parameters[p] = &arm.ParametersParameter{
			Value: v,
		}
	}

	parameters.Parameters["fullDeploy"] = &arm.ParametersParameter{
		Value: d.fullDeploy,
	}

	return parameters
}
