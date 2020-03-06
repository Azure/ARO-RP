package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	uuid "github.com/satori/go.uuid"

	"github.com/Azure/ARO-RP/pkg/util/arm"
)

var apiVersions = map[string]string{
	"authorization": "2018-09-01-preview",
	"compute":       "2019-03-01",
	"dns":           "2018-05-01",
	"documentdb":    "2019-08-01",
	"keyvault":      "2016-10-01",
	"msi":           "2018-11-30",
	"network":       "2019-07-01",
}

const (
	tenantIDHack = "13805ec3-a223-47ad-ad65-8b2baf92c0fb"
)

var (
	tenantUUIDHack = uuid.Must(uuid.FromString(tenantIDHack))
)

type generator struct {
	production bool
}

func newGenerator(production bool) *generator {
	return &generator{
		production: production,
	}
}

// GenerateRPTemplates generates RP templates for production and development
// outputs: database-development.json, rp-development.json, rp-production.json
func GenerateRPTemplates() error {
	for _, i := range []struct {
		templateFile string
		g            *generator
	}{
		{
			templateFile: fileRPDevelopment,
			g:            newGenerator(false),
		},
		{
			templateFile: FileRPProduction,
			g:            newGenerator(true),
		},
	} {
		b, err := json.MarshalIndent(i.g.rpTemplate(), "", "    ")
		if err != nil {
			return err
		}

		// :-(
		b = bytes.ReplaceAll(b, []byte(tenantIDHack), []byte("[subscription().tenantId]"))
		if i.g.production {
			b = bytes.Replace(b, []byte(`"accessPolicies": []`), []byte(`"accessPolicies": "[concat(variables('clustersKeyvaultAccessPolicies'), parameters('extraKeyvaultAccessPolicies'))]"`), 1)
			b = bytes.Replace(b, []byte(`"accessPolicies": []`), []byte(`"accessPolicies": "[concat(variables('serviceKeyvaultAccessPolicies'), parameters('extraKeyvaultAccessPolicies'))]"`), 1)
		}

		b = append(b, byte('\n'))

		err = ioutil.WriteFile(i.templateFile, b, 0666)
		if err != nil {
			return err
		}
	}

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Parameters: map[string]*arm.TemplateParameter{
			"databaseAccountName": {
				Type: "string",
			},
			"databaseName": {
				Type: "string",
			},
		},
	}

	g := newGenerator(false)

	t.Resources = append(t.Resources, g.database("parameters('databaseName')", false)...)

	b, err := json.MarshalIndent(t, "", "    ")
	if err != nil {
		return err
	}

	b = append(b, byte('\n'))

	return ioutil.WriteFile(fileDatabaseDevelopment, b, 0666)
}

// GenerateRPParameterTemplate generates RP parameters file
// output: rp-production-parameters.json
func GenerateRPParameterTemplate() error {
	t := newGenerator(true).rpTemplate()

	p := &arm.Parameters{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentParameters.json#",
		ContentVersion: "1.0.0.0",
		Parameters:     map[string]*arm.ParametersParameter{},
	}

	for name, tp := range t.Parameters {
		param := &arm.ParametersParameter{Value: tp.DefaultValue}
		if param.Value == nil {
			param.Value = ""
		}
		p.Parameters[name] = param
	}

	b, err := json.MarshalIndent(p, "", "    ")
	if err != nil {
		return err
	}

	b = append(b, byte('\n'))

	err = ioutil.WriteFile(fileRPProductionParameters, b, 0666)
	if err != nil {
		return err
	}

	return nil
}

// GenerateNSGTemplates generates Network security group file for development and production
// outputs: rp-development-ngs.json, rp-production-ngs.json
func GenerateNSGTemplates() error {
	for _, i := range []struct {
		templateFile string
		g            *generator
	}{
		{
			templateFile: fileRPDevelopmentNSG,
			g:            newGenerator(false),
		},
		{
			templateFile: FileRPProductionNSG,
			g:            newGenerator(true),
		},
	} {
		t := &arm.Template{
			Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
			ContentVersion: "1.0.0.0",
			Parameters:     map[string]*arm.TemplateParameter{},
		}

		t.Resources = append(t.Resources, i.g.securityGroupRP(), i.g.securityGroupPE())

		if i.g.production {
			t.Resources = append(t.Resources, i.g.managedIdentity())
			t.Outputs = map[string]*arm.Output{
				"rpServicePrincipalId": {
					Type:  "string",
					Value: "[reference(resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', 'rp-identity'), '2018-11-30').principalId]",
				},
			}
		}

		b, err := json.MarshalIndent(t, "", "    ")
		if err != nil {
			return err
		}

		b = append(b, byte('\n'))

		err = ioutil.WriteFile(i.templateFile, b, 0666)
		if err != nil {
			return err
		}

	}
	return nil
}

// GenerateDevelopmentTemplate shared RP template for development
// outputs: env-development.json
func GenerateDevelopmentTemplate() error {
	g := newGenerator(false)

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Parameters:     map[string]*arm.TemplateParameter{},
	}

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

	b, err := json.MarshalIndent(t, "", "    ")
	if err != nil {
		return err
	}

	b = append(b, byte('\n'))

	return ioutil.WriteFile(fileEnvDevelopment, b, 0666)
}

func (g *generator) rpTemplate() *arm.Template {
	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Parameters:     map[string]*arm.TemplateParameter{},
	}

	if g.production {
		t.Variables = map[string]interface{}{
			"clustersKeyvaultAccessPolicies": g.clustersKeyvaultAccessPolicies(),
			"serviceKeyvaultAccessPolicies":  g.serviceKeyvaultAccessPolicies(),
		}
	}

	params := []string{
		"databaseAccountName",
		"domainName",
		"fpServicePrincipalId",
		"keyvaultPrefix",
		"rpServicePrincipalId",
	}
	if g.production {
		params = append(params,
			"extraCosmosDBIPs",
			"extraKeyvaultAccessPolicies",
			"mdmFrontendUrl",
			"mdsdConfigVersion",
			"mdsdEnvironment",
			"pullSecret",
			"rpImage",
			"rpImageAuth",
			"rpMode",
			"sshPublicKey",
			"vmssName",
			"adminApiCaBundle",
			"adminApiClientCertCommonName",
		)
	} else {
		params = append(params,
			"adminObjectId",
		)
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "extraCosmosDBIPs", "rpMode":
			p.DefaultValue = ""
		case "extraKeyvaultAccessPolicies":
			p.Type = "array"
			p.DefaultValue = []mgmtkeyvault.AccessPolicyEntry{}
		case "pullSecret", "rpImageAuth":
			p.Type = "securestring"
		case "keyvaultPrefix":
			p.MaxLength = 24 - max(len(kvClusterSuffix), len(kvServiceSuffix))
		}
		t.Parameters[param] = p
	}

	if g.production {
		t.Resources = append(t.Resources, g.pip(), g.lb(), g.vmss())
	}
	// clustersKeyvault must preceed serviceKeyvault due to terrible bytes.Replace below
	t.Resources = append(t.Resources, g.zone(),
		g.clustersKeyvault(), g.serviceKeyvault(),
		g.rpvnet(), g.pevnet(),
		g.halfPeering("rp-vnet", "rp-pe-vnet-001"),
		g.halfPeering("rp-pe-vnet-001", "rp-vnet"))
	if g.production {
		t.Resources = append(t.Resources, g.cosmosdb("'ARO'")...)
	} else {
		t.Resources = append(t.Resources, g.cosmosdb("pparameters('databaseName')")...)
	}
	t.Resources = append(t.Resources, g.rbac()...)

	return t
}
