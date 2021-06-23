package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func (d *deployer) DeployRP(ctx context.Context) error {
	rpMSI, err := d.userassignedidentities.Get(ctx, d.config.RPResourceGroupName, "aro-rp-"+d.config.Location)
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
	if d.config.Configuration.ARMAPICABundle != nil {
		parameters.Parameters["armApiCaBundle"] = &arm.ParametersParameter{
			Value: base64.StdEncoding.EncodeToString([]byte(*d.config.Configuration.ARMAPICABundle)),
		}
	}
	parameters.Parameters["extraCosmosDBIPs"] = &arm.ParametersParameter{
		Value: strings.Join(d.config.Configuration.ExtraCosmosDBIPs, ","),
	}
	parameters.Parameters["rpImage"] = &arm.ParametersParameter{
		Value: *d.config.Configuration.RPImagePrefix + ":" + d.version,
	}
	parameters.Parameters["rpServicePrincipalId"] = &arm.ParametersParameter{
		Value: rpMSI.PrincipalID.String(),
	}
	parameters.Parameters["vmssName"] = &arm.ParametersParameter{
		Value: d.version,
	}
	parameters.Parameters["keyvaultDNSSuffix"] = &arm.ParametersParameter{
		Value: d.env.Environment().KeyVaultDNSSuffix,
	}
	parameters.Parameters["fpServicePrincipalId"] = &arm.ParametersParameter{
		Value: *d.config.Configuration.FPServicePrincipalID,
	}
	parameters.Parameters["azureCloudName"] = &arm.ParametersParameter{
		Value: d.env.Environment().ActualCloudName,
	}

	for i := 0; i < 2; i++ {
		d.log.Printf("deploying %s", deploymentName)
		err = d.deployments.CreateOrUpdateAndWait(ctx, d.config.RPResourceGroupName, deploymentName, mgmtfeatures.Deployment{
			Properties: &mgmtfeatures.DeploymentProperties{
				Template:   template,
				Mode:       mgmtfeatures.Incremental,
				Parameters: parameters.Parameters,
			},
		})
		if serviceErr, ok := err.(*azure.ServiceError); ok &&
			serviceErr.Code == "DeploymentFailed" &&
			i < 1 {
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

	return d.configureDNS(ctx)
}

func (d *deployer) configureDNS(ctx context.Context) error {
	rpPip, err := d.publicipaddresses.Get(ctx, d.config.RPResourceGroupName, "rp-pip", "")
	if err != nil {
		return err
	}

	portalPip, err := d.publicipaddresses.Get(ctx, d.config.RPResourceGroupName, "portal-pip", "")
	if err != nil {
		return err
	}

	lb, err := d.loadbalancers.Get(ctx, d.config.RPResourceGroupName, "rp-lb-internal", "")
	if err != nil {
		return err
	}
	dbtokenIp := *((*lb.FrontendIPConfigurations)[0].PrivateIPAddress)

	zone, err := d.zones.Get(ctx, d.config.RPResourceGroupName, d.config.Location+"."+*d.config.Configuration.ClusterParentDomainName)
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

	_, err = d.globalrecordsets.CreateOrUpdate(ctx, *d.config.Configuration.GlobalResourceGroupName, *d.config.Configuration.RPParentDomainName, d.config.Location+".admin", mgmtdns.A, mgmtdns.RecordSet{
		RecordSetProperties: &mgmtdns.RecordSetProperties{
			TTL: to.Int64Ptr(3600),
			ARecords: &[]mgmtdns.ARecord{
				{
					Ipv4Address: portalPip.IPAddress,
				},
			},
		},
	}, "", "")
	if err != nil {
		return err
	}

	_, err = d.globalrecordsets.CreateOrUpdate(ctx, *d.config.Configuration.GlobalResourceGroupName, *d.config.Configuration.RPParentDomainName, "dbtoken."+d.config.Location, mgmtdns.A, mgmtdns.RecordSet{
		RecordSetProperties: &mgmtdns.RecordSetProperties{
			TTL: to.Int64Ptr(3600),
			ARecords: &[]mgmtdns.ARecord{
				{
					Ipv4Address: &dbtokenIp,
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
