package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
)

func (g *generator) clusterPredeploy() *arm.Template {
	t := templateStanza()

	params := []string{
		"clusterName",
		"clusterServicePrincipalId",
		"fpServicePrincipalId",
		"fullDeploy",
		"masterAddressPrefix",
		"workerAddressPrefix",
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "fullDeploy":
			p.Type = "bool"
			p.DefaultValue = false
		}
		t.Parameters[param] = p
	}

	t.Resources = append(t.Resources,
		clusterVnet(),
		clusterRouteTable(),
		clusterMasterSubnet(),
		clusterWorkerSubnet(),
		rbac.ResourceRoleAssignment(rbac.RoleContributor, "parameters('clusterServicePrincipalId')", "Microsoft.Network/virtualNetworks", "'dev-vnet'"),
		rbac.ResourceRoleAssignment(rbac.RoleContributor, "parameters('fpServicePrincipalId')", "Microsoft.Network/virtualNetworks", "'dev-vnet'"),
		rbac.ResourceRoleAssignment(rbac.RoleContributor, "parameters('clusterServicePrincipalId')", "Microsoft.Network/routeTables", "concat(parameters('clusterName'), '-rt')"),
		rbac.ResourceRoleAssignment(rbac.RoleContributor, "parameters('fpServicePrincipalId')", "Microsoft.Network/routeTables", "concat(parameters('clusterName'), '-rt')"),
	)

	return t
}

func (g *generator) managedIdentityTemplate() *arm.Template {
	t := templateStanza()

	params := []string{
		"fullDeploy",
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "fullDeploy":
			p.Type = "bool"
			p.DefaultValue = false
		}
		t.Parameters[param] = p
	}

	t.Resources = append(t.Resources,
		g.managedIdentity(),
	)

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
			"acrResourceId",
			"adminApiCaBundle",
			"adminApiClientCertCommonName",
			"extraCosmosDBIPs",
			"fullDeploy",
			"mdmFrontendUrl",
			"mdsdConfigVersion",
			"mdsdEnvironment",
			"rpImage",
			"rpMode",
			"sshPublicKey",
			"subscriptionResourceGroupName",
			"vmssName",
			"vmSize",
		)
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "extraCosmosDBIPs", "rpMode":
			p.DefaultValue = ""
		case "fullDeploy":
			p.Type = "bool"
			p.DefaultValue = false
		case "vmSize":
			p.DefaultValue = "Standard_D2s_v3"
		}
		t.Parameters[param] = p
	}

	if g.production {
		t.Resources = append(t.Resources, g.pip(), g.lb(), g.vmss(),
			g.lbAlert(30.0, 2, "rp-availability-alert", "PT5M", "PT5M", "DipAvailability"), // triggers on all 3 RPs being down for 3.5min, can't be >=0.3 due to deploys going down to 32% at times.
			g.lbAlert(67.0, 3, "rp-degraded-alert", "PT15M", "PT6H", "DipAvailability"),    // 1/3 backend down for 1h or 2/3 down for 3h in the last 6h
			g.lbAlert(33.0, 2, "rp-vnet-alert", "PT5M", "PT5M", "VipAvailability"))         // this will trigger only if the Azure network infrastructure between the loadBalancers and VMs is down for 3.5min
		// more on alerts https://msazure.visualstudio.com/AzureRedHatOpenShift/_wiki/wikis/ARO.wiki/53765/WIP-Alerting
		t.Resources = append(t.Resources, g.billingContributorRbac()...)
	}

	t.Resources = append(t.Resources, g.zone(),
		g.rpvnet(), g.pevnet(),
		g.halfPeering("rp-vnet", "rp-pe-vnet-001"),
		g.halfPeering("rp-pe-vnet-001", "rp-vnet"))
	t.Resources = append(t.Resources, g.cosmosdb()...)
	t.Resources = append(t.Resources, g.rbac()...)

	return t
}

func (g *generator) rpGlobalTemplate() *arm.Template {
	t := templateStanza()

	params := []string{
		"acrResourceId",
		"fullDeploy",
		"fpServicePrincipalId",
		"rpServicePrincipalId",
		"rpVersionStorageAccountName",
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "fullDeploy":
			p.Type = "bool"
			p.DefaultValue = false
		}
		t.Parameters[param] = p
	}
	t.Resources = append(t.Resources,
		g.acrRbac()...,
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
		"fullDeploy",
		"location",
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "fullDeploy":
			p.Type = "bool"
			p.DefaultValue = false
		}
		t.Parameters[param] = p
	}
	t.Resources = append(t.Resources,
		g.acrReplica(),
	)

	return t
}

func (g *generator) rpGlobalSubscriptionTemplate() *arm.Template {
	t := templateStanza()

	params := []string{
		"fullDeploy",
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "fullDeploy":
			p.Type = "bool"
			p.DefaultValue = false
		}
		t.Parameters[param] = p
	}

	t.Resources = append(t.Resources,
		g.roleDefinitionTokenContributor(),
	)

	return t
}

func (g *generator) rpSubscriptionTemplate() *arm.Template {
	t := templateStanza()

	params := []string{
		"fullDeploy",
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "fullDeploy":
			p.Type = "bool"
			p.DefaultValue = false
		}
		t.Parameters[param] = p
	}

	t.Resources = append(t.Resources, g.actionGroup("rp-health-ag", "rphealth"))

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
			"fullDeploy",
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
		case "fullDeploy":
			p.Type = "bool"
			p.DefaultValue = false
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
		g.devCIPool(),
		g.proxyVmss())

	for _, param := range []string{
		"ciAzpToken",
		"ciCapacity",
		"ciPoolName",
		"ciDeployTooling",
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
		case "ciDeployTooling":
			typ = "bool"
			defaultValue = false
		case "ciCapacity":
			typ = "int"
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
	b = bytes.ReplaceAll(b, []byte(`"capacity": 1337`), []byte(`"capacity": "[int(parameters('ciCapacity'))]"`))
	if g.production {
		b = bytes.Replace(b, []byte(`"accessPolicies": []`), []byte(`"accessPolicies": "[concat(variables('clusterKeyvaultAccessPolicies'), parameters('extraClusterKeyvaultAccessPolicies'))]"`), 1)
		b = bytes.Replace(b, []byte(`"accessPolicies": []`), []byte(`"accessPolicies": "[concat(variables('serviceKeyvaultAccessPolicies'), parameters('extraServiceKeyvaultAccessPolicies'))]"`), 1)
		b = bytes.Replace(b, []byte(`"sourceAddressPrefixes": []`), []byte(`"sourceAddressPrefixes": "[parameters('rpNsgSourceAddressPrefixes')]"`), 1)
	}

	return append(b, byte('\n')), nil
}

func (g *generator) conditionStanza(parameterName string) interface{} {
	if g.production {
		return "[parameters('" + parameterName + "')]"
	}

	return nil
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
