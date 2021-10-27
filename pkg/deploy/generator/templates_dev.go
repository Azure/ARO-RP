package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func (g *generator) devDatabaseTemplate() *arm.Template {
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

func (g *generator) devSharedTemplate() *arm.Template {
	t := templateStanza()

	t.Resources = append(t.Resources,
		g.devVPNPip(),
		g.devVnet(),
		g.devVPN(),
		g.devCIPool(),
		g.devDiskEncryptionKeyvault(),
		g.devDiskEncryptionKey(),
		g.devDiskEncryptionSet(),
		g.devProxyVMSS())

	for _, param := range []string{
		"azureServicePrincipalId",
		"ciAzpToken",
		"ciCapacity",
		"ciPoolName",
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
		case "ciAzpToken", "ciPoolName":
			defaultValue = ""
		case "ciCapacity":
			typ = "int"
			defaultValue = 0
		case "proxyImageAuth", "proxyKey":
			typ = "securestring"
		case "publicIPAddressAllocationMethod":
			defaultValue = "Static"
		case "publicIPAddressSkuName":
			defaultValue = "Standard"
		case "vpnCACertificate":
			defaultValue = ""
		}
		t.Parameters[param] = &arm.TemplateParameter{
			Type:         typ,
			DefaultValue: defaultValue,
		}
	}

	return t
}
