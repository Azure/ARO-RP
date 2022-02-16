package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func (g *generator) gatewayManagedIdentityTemplate() *arm.Template {
	t := templateStanza()

	t.Resources = append(t.Resources,
		g.gatewayManagedIdentity(),
	)

	return t
}

func (g *generator) gatewayTemplate() *arm.Template {
	t := templateStanza()

	params := []string{
		"acrResourceId",
		"azureCloudName",
		"azureSecPackVSATenantId",
		"databaseAccountName",
		"dbtokenClientId",
		"dbtokenUrl",
		"gatewayDomains",
		"gatewayFeatures",
		"gatewayMdsdConfigVersion",
		"gatewayServicePrincipalId",
		"gatewayStorageAccountDomain",
		"gatewayVmSize",
		"gatewayVmssCapacity",
		"keyvaultDNSSuffix",
		"keyvaultPrefix",
		"mdmFrontendUrl",
		"mdsdEnvironment",
		"nonZonalRegions",
		"rpImage",
		"rpMdmAccount",
		"rpMdsdAccount",
		"rpMdsdNamespace",
		"rpResourceGroupName",
		"rpServicePrincipalId",
		"sshPublicKey",
		"vmssCleanupEnabled",
		"vmssName",
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "gatewayDomains",
			"gatewayFeatures":
			p.DefaultValue = ""
		case "gatewayVmSize":
			p.DefaultValue = "Standard_D4s_v3"
		case "gatewayVmssCapacity":
			p.Type = "int"
			p.DefaultValue = 3
		case "vmssCleanupEnabled":
			p.Type = "bool"
			p.DefaultValue = true
		case "nonZonalRegions":
			p.Type = "array"
			p.DefaultValue = []string{
				"eastasia",
				"centralindia",
				"centraluseuap",
				"koreacentral",
				"switzerlandnorth",
				"northcentralus",
				"uaenorth",
			}
		}
		t.Parameters[param] = p
	}

	t.Resources = append(t.Resources,
		g.gatewayStorageAccount(),
		g.gatewayLB(),
		g.gatewayPLS(),
		g.gatewayVMSS(),
		// TODO: use rpLBAlert() to define Azure monitoring alerts around the readiness of the gateway ILB
	)

	t.Resources = append(t.Resources,
		g.virtualNetworkPeering("gateway-vnet/peering-rp-vnet", "[resourceId(parameters('rpResourceGroupName'), 'Microsoft.Network/virtualNetworks', 'rp-vnet')]"),
	)

	t.Resources = append(t.Resources, g.gatewayRBAC()...)

	return t
}

func (g *generator) gatewayParameters() *arm.Parameters {
	t := g.gatewayTemplate()
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

func (g *generator) gatewayPredeployTemplate() *arm.Template {
	t := templateStanza()

	if g.production {
		t.Variables = map[string]interface{}{
			"gatewayKeyvaultAccessPolicies": g.gatewayKeyvaultAccessPolicies(),
		}
	}

	params := []string{
		"deployNSGs",
		"extraGatewayKeyvaultAccessPolicies",
		"gatewayServicePrincipalId",
		"keyvaultPrefix",
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "deployNSGs":
			p.Type = "bool"
			p.DefaultValue = false
		case "extraGatewayKeyvaultAccessPolicies":
			p.Type = "array"
			p.DefaultValue = []interface{}{}
		case "keyvaultPrefix":
			p.MaxLength = 24 - len(env.GatewayKeyvaultSuffix)
		}
		t.Parameters[param] = p
	}

	t.Resources = append(t.Resources,
		g.gatewaySecurityGroup(),
		g.gatewayVnet(),
		g.gatewayKeyvault(),
	)

	return t
}

func (g *generator) gatewayPredeployParameters() *arm.Parameters {
	t := g.gatewayPredeployTemplate()
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
