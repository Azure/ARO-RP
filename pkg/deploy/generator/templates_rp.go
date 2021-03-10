package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
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
			"billingE2EStorageAccountId",
			"billingServicePrincipalId",
			"clusterMdsdConfigVersion",
			"encryptionAtHost",
			"extraCosmosDBIPs",
			"fpClientId",
			"keyvaultPrefix",
			"mdmFrontendUrl",
			"mdsdEnvironment",
			"portalAccessGroupIds",
			"portalClientId",
			"portalElevatedGroupIds",
			"rpFeatures",
			"rpImage",
			"rpMdsdConfigVersion",
			"rpParentDomainName",
			"sshPublicKey",
			"storageAccountDomain",
			"subscriptionResourceGroupName",
			"vmssName",
			"vmSize",
		)
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "encryptionAtHost":
			p.Type = "bool"
		case "billingServicePrincipalId",
			"billingE2EStorageAccountId",
			"extraCosmosDBIPs",
			"rpFeatures":
			p.DefaultValue = ""
		case "vmSize":
			p.DefaultValue = "Standard_D2s_v3"
		}
		t.Parameters[param] = p
	}

	if g.production {
		t.Resources = append(t.Resources,
			g.publicIPAddress("rp-pip"),
			g.publicIPAddress("portal-pip"),
			g.rpLB(),
			g.rpVMSS(),
			g.rpStorageAccount(),
			g.rpLBAlert(30.0, 2, "rp-availability-alert", "PT5M", "PT15M", "DipAvailability"), // triggers on all 3 RPs being down for 10min, can't be >=0.3 due to deploys going down to 32% at times.
			g.rpLBAlert(67.0, 3, "rp-degraded-alert", "PT15M", "PT6H", "DipAvailability"),     // 1/3 backend down for 1h or 2/3 down for 3h in the last 6h
			g.rpLBAlert(33.0, 2, "rp-vnet-alert", "PT5M", "PT5M", "VipAvailability"))          // this will trigger only if the Azure network infrastructure between the loadBalancers and VMs is down for 3.5min
		// more on alerts https://msazure.visualstudio.com/AzureRedHatOpenShift/_wiki/wikis/ARO.wiki/53765/WIP-Alerting
		t.Resources = append(t.Resources, g.rpBillingContributorRbac()...)
	}

	t.Resources = append(t.Resources, g.rpDNSZone(),
		g.rpVnet(), g.rpPEVnet(),
		g.virtualNetworkPeering("rp-vnet", "rp-pe-vnet-001"),
		g.virtualNetworkPeering("rp-pe-vnet-001", "rp-vnet"))
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
			"extraPortalKeyvaultAccessPolicies",
			"extraServiceKeyvaultAccessPolicies",
			"rpNsgSourceAddressPrefixes",
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
			"extraPortalKeyvaultAccessPolicies",
			"extraServiceKeyvaultAccessPolicies":
			p.Type = "array"
			p.DefaultValue = []interface{}{}
		case "rpNsgSourceAddressPrefixes":
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
		g.rpClusterKeyvault(),
		g.rpPortalKeyvault(),
		g.rpServiceKeyvault(),
	)

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
