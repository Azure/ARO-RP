package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"

	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func (g *generator) managedIdentityTemplate() *arm.Template {
	t := templateStanza()

	t.Resources = append(t.Resources,
		g.managedIdentity(),
	)

	t.Outputs = map[string]*arm.Output{
		"rpServicePrincipalId": {
			Type:  "string",
			Value: "[reference(resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', concat('aro-rp-', resourceGroup().location)), '2018-11-30').principalId]",
		},
	}

	return t
}

func (g *generator) rpTemplate() *arm.Template {
	t := templateStanza()

	params := []string{
		"databaseAccountName",
		"domainName",
		"fpServicePrincipalId",
		"rpServicePrincipalId",
	}
	if g.production {
		params = append(params,
			"adminApiCaBundle",
			"adminApiClientCertCommonName",
			"extraCosmosDBIPs",
			"mdmFrontendUrl",
			"mdsdConfigVersion",
			"mdsdEnvironment",
			"acrResourceId",
			"rpImage",
			"rpMode",
			"sshPublicKey",
			"vmssName",
		)
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "extraCosmosDBIPs", "rpMode":
			p.DefaultValue = ""
		}
		t.Parameters[param] = p
	}

	if g.production {
		t.Resources = append(t.Resources, g.pip(), g.lb(), g.vmss())
	}

	t.Resources = append(t.Resources, g.zone(),
		g.rpvnet(), g.pevnet(),
		g.halfPeering("rp-vnet", "rp-pe-vnet-001"),
		g.halfPeering("rp-pe-vnet-001", "rp-vnet"))
	t.Resources = append(t.Resources, g.cosmosdb()...)
	t.Resources = append(t.Resources, g.rbac()...)

	t.Outputs = map[string]*arm.Output{
		"rp-nameServers": {
			Type:  "array",
			Value: "[reference(resourceId('Microsoft.Network/dnsZones', parameters('domainName')), '2018-05-01').nameServers]",
		},
		"rp-pip-ipAddress": {
			Type:  "string",
			Value: "[reference(resourceId('Microsoft.Network/publicIPAddresses', 'rp-pip'), '2019-07-01').ipAddress]",
		},
	}

	return t
}

func (g *generator) rpGlobalTemplate() *arm.Template {
	t := templateStanza()

	params := []string{
		"acrResourceId",
		"fpServicePrincipalId",
		"location",
		"rpServicePrincipalId",
	}

	for _, param := range params {
		t.Parameters[param] = &arm.TemplateParameter{Type: "string"}
	}

	t.Resources = append(t.Resources,
		g.acrReplica(),
	)

	t.Resources = append(t.Resources,
		g.acrRbac()...,
	)

	return t
}

func (g *generator) rpGlobalSubscriptionTemplate() *arm.Template {
	t := templateStanza()

	t.Resources = append(t.Resources,
		g.roleDefinitionTokenContributor(),
	)

	return t
}

func (g *generator) databaseTemplate() *arm.Template {
	t := templateStanza()

	t.Resources = append(t.Resources,
		g.database("parameters('databaseName')", false)...)

	t.Parameters = map[string]*arm.TemplateParameter{
		"databaseAccountName": {
			Type: "string",
		},
		"databaseName": {
			Type: "string",
		},
	}

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

func (g *generator) preDeployTemplate() *arm.Template {
	t := templateStanza()

	if g.production {
		t.Variables = map[string]interface{}{
			"clusterKeyvaultAccessPolicies": g.clusterKeyvaultAccessPolicies(),
			"serviceKeyvaultAccessPolicies": g.serviceKeyvaultAccessPolicies(),
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
		case "extraClusterKeyvaultAccessPolicies", "extraServiceKeyvaultAccessPolicies":
			p.Type = "array"
			p.DefaultValue = []interface{}{}
		case "rpNsgSourceAddressPrefixes":
			p.Type = "array"
			p.DefaultValue = []string{}
		case "keyvaultPrefix":
			p.MaxLength = 24 - max(len(kvClusterSuffix), len(kvServiceSuffix))
		}
		t.Parameters[param] = p
	}

	t.Resources = append(t.Resources,
		g.securityGroupRP(),
		g.securityGroupPE(),
		// clustersKeyvault must preceed serviceKeyvault due to terrible
		// bytes.Replace in templateFixup
		g.clustersKeyvault(),
		g.serviceKeyvault(),
	)

	return t
}

func (g *generator) rpPreDeployParameters() *arm.Parameters {
	t := g.preDeployTemplate()
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

func (g *generator) sharedDevelopmentEnvTemplate() *arm.Template {
	t := templateStanza()

	t.Resources = append(t.Resources,
		g.devVpnPip(),
		g.devVnet(),
		g.devVPN(),
		g.proxyVmss())

	for _, param := range []string{
		"proxyCert",
		"proxyClientCert",
		"proxyDomainNameLabel",
		"proxyImage",
		"proxyImageAuth",
		"proxyKey",
		"publicIPAddressAllocationMethod",
		"publicIPAddressSkuName",
		"sshPublicKey",
		"vpnCACertificate",
	} {
		typ := "string"
		var defaultValue interface{}
		switch param {
		case "proxyImageAuth", "proxyKey":
			typ = "securestring"
		case "publicIPAddressAllocationMethod":
			defaultValue = "Static"
		case "publicIPAddressSkuName":
			defaultValue = "Standard"
		}
		t.Parameters[param] = &arm.TemplateParameter{
			Type:         typ,
			DefaultValue: defaultValue,
		}
	}

	return t
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (g *generator) templateFixup(t *arm.Template) ([]byte, error) {
	b, err := json.MarshalIndent(t, "", "    ")
	if err != nil {
		return nil, err
	}

	// :-(
	b = bytes.ReplaceAll(b, []byte(tenantIDHack), []byte("[subscription().tenantId]"))
	if g.production {
		b = bytes.Replace(b, []byte(`"accessPolicies": []`), []byte(`"accessPolicies": "[concat(variables('clusterKeyvaultAccessPolicies'), parameters('extraClusterKeyvaultAccessPolicies'))]"`), 1)
		b = bytes.Replace(b, []byte(`"accessPolicies": []`), []byte(`"accessPolicies": "[concat(variables('serviceKeyvaultAccessPolicies'), parameters('extraServiceKeyvaultAccessPolicies'))]"`), 1)
		b = bytes.Replace(b, []byte(`"sourceAddressPrefixes": []`), []byte(`"sourceAddressPrefixes": "[parameters('rpNsgSourceAddressPrefixes')]"`), 1)
	}

	return append(b, byte('\n')), nil
}

func templateStanza() *arm.Template {
	return &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Parameters:     map[string]*arm.TemplateParameter{},
	}
}

func parametersStanza() *arm.Parameters {
	return &arm.Parameters{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentParameters.json#",
		ContentVersion: "1.0.0.0",
		Parameters:     map[string]*arm.ParametersParameter{},
	}
}
