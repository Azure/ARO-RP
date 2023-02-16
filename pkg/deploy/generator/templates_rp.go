package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func (g *generator) rpManagedIdentityTemplate() *arm.Template {
	t := templateStanza()

	t.Resources = append(t.Resources,
		g.rpManagedIdentity(),
	)

	return t
}

func (g *generator) rpTemplate() *arm.Template {
	t := templateStanza()

	params := []string{
		"clusterParentDomainName",
		"databaseAccountName",
		"fpServicePrincipalId",
		"rpServicePrincipalId",
	}
	if g.production {
		params = append(params,
			"acrResourceId",
			"adminApiCaBundle",
			"adminApiClientCertCommonName",
			"armApiCaBundle",
			"armApiClientCertCommonName",
			"armClientId",
			"azureCloudName",
			"azureSecPackQualysUrl",
			"azureSecPackVSATenantId",
			"billingE2EStorageAccountId",
			"billingServicePrincipalId",
			"clusterMdmAccount",
			"clusterMdsdAccount",
			"clusterMdsdConfigVersion",
			"clusterMdsdNamespace",
			"dbtokenClientId",
			"disableCosmosDBFirewall",
			"fluentbitImage",
			"fpClientId",
			"fpServicePrincipalId",
			"ipRules",
			"keyvaultPrefix",
			"keyvaultDNSSuffix",
			"gatewayDomains",
			"gatewayResourceGroupName",
			"gatewayServicePrincipalId",
			"ipRules",
			"mdmFrontendUrl",
			"mdsdEnvironment",
			"nonZonalRegions",
			"portalAccessGroupIds",
			"portalClientId",
			"portalElevatedGroupIds",
			"rpFeatures",
			"rpImage",
			"rpMdmAccount",
			"rpMdsdAccount",
			"rpMdsdConfigVersion",
			"rpMdsdNamespace",
			"rpParentDomainName",
			"rpVmssCapacity",
			"sshPublicKey",
			"storageAccountDomain",
			"subscriptionResourceGroupName",
			"vmSize",
			"vmssCleanupEnabled",
			"vmssName",

			// TODO: Replace with Live Service Configuration in KeyVault
			"clustersInstallViaHive",
			"clusterDefaultInstallerPullspec",
			"clustersAdoptByHive",
		)
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "disableCosmosDBFirewall":
			p.Type = "bool"
			p.DefaultValue = false
		case "ipRules":
			p.Type = "array"
		case "armApiCaBundle",
			"armApiClientCertCommonName",
			"armClientId",
			"billingServicePrincipalId",
			"billingE2EStorageAccountId",
			"gatewayDomains",
			"rpFeatures":
			p.DefaultValue = ""
		case "vmSize":
			p.DefaultValue = "Standard_D2s_v3"
		case "vmssCleanupEnabled":
			p.Type = "bool"
			p.DefaultValue = true
		case "rpVmssCapacity":
			p.Type = "int"
			p.DefaultValue = 3
		case "nonZonalRegions":
			p.Type = "array"
			p.DefaultValue = []string{
				"eastasia",
				"centralindia",
				"centraluseuap",
				"koreacentral",
				"southcentralus",
				"canadacentral",
				"germanywestcentral",
				"norwayeast",
				"switzerlandnorth",
				"brazilsouth",
				"southafricanorth",
				"northcentralus",
				"uaenorth",
				"westus",
			}

		// TODO: Replace with Live Service Configuration in KeyVault
		case "clustersInstallViaHive",
			"clustersAdoptByHive",
			"clusterDefaultInstallerPullspec":
			p.DefaultValue = ""
		}
		t.Parameters[param] = p
	}

	if g.production {
		t.Variables = map[string]interface{}{
			"rpCosmoDbVirtualNetworkRules": to.StringPtr("createArray(" +
				" createObject('id', resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet'))," +
				" createObject('id', resourceId(parameters('gatewayResourceGroupName'), 'Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet'))" +
				")"),
			"rpCosmoDbVirtualNetworkRulesVmDeploy": to.StringPtr("createArray(" +
				" createObject('id', resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet'))," +
				" createObject('id', resourceId(parameters('gatewayResourceGroupName'), 'Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet'))" +
				" createObject('id', resourceId('Microsoft.Network/virtualNetworks/subnets', 'aks-net', 'ClusterSubnet-001'))," +
				// TODO: AKS Sharding: add rules for additional AKS shards for this RP instance. Currently only shard 1, which has subnet ClusterSubnet-001, is set above.
				// AKS subnet design: https://docs.google.com/document/d/1gTGSW5S4uN1vB2hqVFKYr-qp6n62WbkdQMrKg-qvPbE
				")"),
		}

		t.Resources = append(t.Resources,
			g.publicIPAddress("rp-pip"),
			g.publicIPAddress("portal-pip"),
			g.rpLB(),
			g.rpLBInternal(),
			g.rpVMSS(),
			g.rpStorageAccount(),
			g.rpLBAlert(30.0, 2, "rp-availability-alert", "PT5M", "PT15M", "DipAvailability"), // triggers on all 3 RPs being down for 10min, can't be >=0.3 due to deploys going down to 32% at times.
			g.rpLBAlert(67.0, 3, "rp-degraded-alert", "PT15M", "PT6H", "DipAvailability"),     // 1/3 backend down for 1h or 2/3 down for 3h in the last 6h
			g.rpLBAlert(33.0, 2, "rp-vnet-alert", "PT5M", "PT5M", "VipAvailability"))          // this will trigger only if the Azure network infrastructure between the loadBalancers and VMs is down for 3.5min
		// more on alerts https://msazure.visualstudio.com/AzureRedHatOpenShift/_wiki/wikis/ARO.wiki/53765/WIP-Alerting
		t.Resources = append(t.Resources, g.rpBillingContributorRbac()...)

		t.Resources = append(t.Resources,
			g.virtualNetworkPeering("rp-vnet/peering-gateway-vnet", "[resourceId(parameters('gatewayResourceGroupName'), 'Microsoft.Network/virtualNetworks', 'gateway-vnet')]", false, false, nil),
		)
	}

	t.Resources = append(t.Resources, g.rpDNSZone(),
		g.virtualNetworkPeering("rp-vnet/peering-rp-pe-vnet-001", "[resourceId('Microsoft.Network/virtualNetworks', 'rp-pe-vnet-001')]", false, false, nil),
		g.virtualNetworkPeering("rp-pe-vnet-001/peering-rp-vnet", "[resourceId('Microsoft.Network/virtualNetworks', 'rp-vnet')]", false, false, nil))
	t.Resources = append(t.Resources, g.rpCosmosDB()...)
	t.Resources = append(t.Resources, g.rpRBAC()...)

	return t
}

func (g *generator) rpGlobalTemplate() *arm.Template {
	t := templateStanza()

	params := []string{
		"acrLocationOverride",
		"acrResourceId",
		"fpServicePrincipalId",
		"clusterParentDomainName",
		"gatewayServicePrincipalId",
		"rpParentDomainName",
		"rpServicePrincipalId",
		"rpVersionStorageAccountName",
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "acrLocationOverride":
			p.DefaultValue = ""
		}
		t.Parameters[param] = p
	}
	t.Resources = append(t.Resources,
		g.rpACR(),
		g.rpParentDNSZone(),
		g.rpClusterParentDNSZone(),
	)
	t.Resources = append(t.Resources,
		g.rpACRRBAC()...,
	)
	t.Resources = append(t.Resources,
		g.rpVersionStorageAccount()...,
	)

	return t
}

func (g *generator) rpGlobalACRReplicationTemplate() *arm.Template {
	t := templateStanza()

	params := []string{
		"acrResourceId",
		"location",
	}

	for _, param := range params {
		t.Parameters[param] = &arm.TemplateParameter{Type: "string"}
	}
	t.Resources = append(t.Resources,
		g.rpACRReplica(),
	)

	return t
}

func (g *generator) rpGlobalSubscriptionTemplate() *arm.Template {
	t := templateStanza()

	t.Resources = append(t.Resources,
		g.rpRoleDefinitionTokenContributor(),
	)

	return t
}

func (g *generator) rpSubscriptionTemplate() *arm.Template {
	t := templateStanza()

	t.Resources = append(t.Resources, g.actionGroup("rp-health-ag", "rphealth"))

	return t
}

func (g *generator) rpParameters() *arm.Parameters {
	t := g.rpTemplate()
	p := parametersStanza()

	for name, tp := range t.Parameters {
		param := &arm.ParametersParameter{Value: tp.DefaultValue}
		if param.Value == nil {
			param.Value = ""
		}
		p.Parameters[name] = param
	}

	return p
}

func (g *generator) rpPredeployTemplate() *arm.Template {
	t := templateStanza()

	if g.production {
		t.Variables = map[string]interface{}{
			"clusterKeyvaultAccessPolicies": g.rpClusterKeyvaultAccessPolicies(),
			"dbTokenKeyvaultAccessPolicies": g.rpDBTokenKeyvaultAccessPolicies(),
			"portalKeyvaultAccessPolicies":  g.rpPortalKeyvaultAccessPolicies(),
			"serviceKeyvaultAccessPolicies": g.rpServiceKeyvaultAccessPolicies(),
		}
	}

	params := []string{
		"keyvaultPrefix",
		"rpServicePrincipalId",
		"fpServicePrincipalId",
	}

	if g.production {
		params = append(params,
			"deployNSGs",
			"extraClusterKeyvaultAccessPolicies",
			"extraDBTokenKeyvaultAccessPolicies",
			"extraPortalKeyvaultAccessPolicies",
			"extraServiceKeyvaultAccessPolicies",
			"gatewayResourceGroupName",
			"rpNsgPortalSourceAddressPrefixes",
		)
	} else {
		params = append(params,
			"adminObjectId",
		)
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "deployNSGs":
			p.Type = "bool"
			p.DefaultValue = false
		case "extraClusterKeyvaultAccessPolicies",
			"extraDBTokenKeyvaultAccessPolicies",
			"extraPortalKeyvaultAccessPolicies",
			"extraServiceKeyvaultAccessPolicies":
			p.Type = "array"
			p.DefaultValue = []interface{}{}
		case "rpNsgPortalSourceAddressPrefixes":
			p.Type = "array"
			p.DefaultValue = []string{}
		case "keyvaultPrefix":
			p.MaxLength = 24 - max(len(env.ClusterKeyvaultSuffix), len(env.ServiceKeyvaultSuffix), len(env.PortalKeyvaultSuffix))
		}
		t.Parameters[param] = p
	}

	t.Resources = append(t.Resources,
		g.rpSecurityGroup(),
		g.rpPESecurityGroup(),
		g.rpVnet(),
		g.rpPEVnet(),
		g.rpClusterKeyvault(),
		g.rpDBTokenKeyvault(),
		g.rpPortalKeyvault(),
		g.rpServiceKeyvault(),
		g.rpServiceKeyvaultDynamic(),
	)

	if g.production {
		t.Resources = append(t.Resources,
			g.rpSecurityGroupForPortalSourceAddressPrefixes(),
		)
	}

	return t
}

func (g *generator) rpPredeployParameters() *arm.Parameters {
	t := g.rpPredeployTemplate()
	p := parametersStanza()

	for name, tp := range t.Parameters {
		param := &arm.ParametersParameter{Value: tp.DefaultValue}
		if param.Value == nil {
			param.Value = ""
		}
		p.Parameters[name] = param
	}

	return p
}
