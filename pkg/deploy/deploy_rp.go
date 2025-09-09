package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"encoding/json"

	mgmtdocumentdb "github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2021-01-15/documentdb"
	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	"github.com/Azure/ARO-RP/pkg/deploy/assets"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func (d *deployer) DeployRP(ctx context.Context) error {
	rpMSI, err := d.userassignedidentities.Get(ctx, d.config.RPResourceGroupName, "aro-rp-"+d.config.Location)
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

	deploymentName := "rp-production-" + d.version

	asset, err := assets.EmbeddedFiles.ReadFile(generator.FileRPProduction)
	if err != nil {
		return err
	}

	var template map[string]interface{}
	err = json.Unmarshal(asset, &template)
	if err != nil {
		return err
	}

	// Special cases where the config isn't marshalled into the ARM template parameters cleanly
	parameters := d.getParameters(template["parameters"].(map[string]interface{}))
	parameters.Parameters["adminApiCaBundle"] = &arm.ParametersParameter{
		Value: base64.StdEncoding.EncodeToString([]byte(*d.config.Configuration.AdminAPICABundle)),
	}
	if d.config.Configuration.ARMAPICABundle != nil {
		parameters.Parameters["armApiCaBundle"] = &arm.ParametersParameter{
			Value: base64.StdEncoding.EncodeToString([]byte(*d.config.Configuration.ARMAPICABundle)),
		}
	}
	ipRules := d.convertToIPAddressOrRange(d.config.Configuration.ExtraCosmosDBIPs)
	parameters.Parameters["ipRules"] = &arm.ParametersParameter{
		Value: ipRules,
	}
	parameters.Parameters["gatewayResourceGroupName"] = &arm.ParametersParameter{
		Value: d.config.GatewayResourceGroupName,
	}
	parameters.Parameters["gatewayServicePrincipalId"] = &arm.ParametersParameter{
		Value: gwMSI.PrincipalID.String(),
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
	parameters.Parameters["vmssIpTags"] = &arm.ParametersParameter{
		Value: d.config.Configuration.VmssIpTags,
	}
	parameters.Parameters["vmssIpTagsDisabledRegions"] = &arm.ParametersParameter{
		Value: d.config.Configuration.VmssIpTagsDisabledRegions,
	}
	parameters.Parameters["keyvaultDNSSuffix"] = &arm.ParametersParameter{
		Value: d.env.Environment().KeyVaultDNSSuffix,
	}
	parameters.Parameters["azureCloudName"] = &arm.ParametersParameter{
		Value: d.env.Environment().ActualCloudName,
	}
	parameters.Parameters["globalDevopsServicePrincipalId"] = &arm.ParametersParameter{
		Value: globalDevopsMSI.PrincipalID.String(),
	}
	if d.config.Configuration.CosmosDB != nil {
		parameters.Parameters["cosmosDB"] = &arm.ParametersParameter{
			Value: map[string]int{
				"standardProvisionedThroughput": d.config.Configuration.CosmosDB.StandardProvisionedThroughput,
				"portalProvisionedThroughput":   d.config.Configuration.CosmosDB.PortalProvisionedThroughput,
				"gatewayProvisionedThroughput":  d.config.Configuration.CosmosDB.GatewayProvisionedThroughput,
			},
		}
	}

	err = d.deploy(ctx, d.config.RPResourceGroupName, deploymentName, rpVMSSPrefix+d.version,
		mgmtfeatures.Deployment{
			Properties: &mgmtfeatures.DeploymentProperties{
				Template:   template,
				Mode:       mgmtfeatures.Incremental,
				Parameters: parameters.Parameters,
			},
		},
	)
	if err != nil {
		return err
	}

	return d.configureDNS(ctx)
}

func (d *deployer) configureDNS(ctx context.Context) error {
	rpPIP, err := d.publicipaddresses.Get(ctx, d.config.RPResourceGroupName, "rp-pip", nil)
	if err != nil {
		return err
	}

	portalPIP, err := d.publicipaddresses.Get(ctx, d.config.RPResourceGroupName, "portal-pip", nil)
	if err != nil {
		return err
	}

	zone, err := d.zones.Get(ctx, d.config.RPResourceGroupName, d.config.Location+"."+*d.config.Configuration.ClusterParentDomainName)
	if err != nil {
		return err
	}

	_, err = d.globalrecordsets.CreateOrUpdate(ctx, *d.config.Configuration.GlobalResourceGroupName, *d.config.Configuration.RPParentDomainName, "rp."+d.config.Location, mgmtdns.A, mgmtdns.RecordSet{
		RecordSetProperties: &mgmtdns.RecordSetProperties{
			TTL: pointerutils.ToPtr(int64(3600)),
			ARecords: &[]mgmtdns.ARecord{
				{
					Ipv4Address: rpPIP.Properties.IPAddress,
				},
			},
		},
	}, "", "")
	if err != nil {
		return err
	}

	_, err = d.globalrecordsets.CreateOrUpdate(ctx, *d.config.Configuration.GlobalResourceGroupName, *d.config.Configuration.RPParentDomainName, d.config.Location+".admin", mgmtdns.A, mgmtdns.RecordSet{
		RecordSetProperties: &mgmtdns.RecordSetProperties{
			TTL: pointerutils.ToPtr(int64(3600)),
			ARecords: &[]mgmtdns.ARecord{
				{
					Ipv4Address: portalPIP.Properties.IPAddress,
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
			TTL:       pointerutils.ToPtr(int64(3600)),
			NsRecords: &nsRecords,
		},
	}, "", "")
	return err
}

func (d *deployer) convertToIPAddressOrRange(ipSlice []string) []mgmtdocumentdb.IPAddressOrRange {
	ips := []mgmtdocumentdb.IPAddressOrRange{}
	for _, v := range ipSlice {
		ips = append(ips, mgmtdocumentdb.IPAddressOrRange{IPAddressOrRange: pointerutils.ToPtr(v)})
	}
	return ips
}
