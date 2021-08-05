package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"
	"regexp"

	"github.com/gofrs/uuid"

	"github.com/Azure/ARO-RP/pkg/util/arm"
)

const (
	tenantIDHack            = "13805ec3-a223-47ad-ad65-8b2baf92c0fb"
	clusterAccessPolicyHack = "e1992efe-4835-46cf-8c08-d8b8451044b8"
	dbTokenAccessPolicyHack = "bb6c76fd-76ea-43c9-8ee3-ca568ae1c226"
	portalAccessPolicyHack  = "e5e11dae-7c49-4118-9628-e0afa4d6a502"
	serviceAccessPolicyHack = "533a94d0-d6c2-4fca-9af1-374aa6493468"
)

var (
	tenantUUIDHack = uuid.Must(uuid.FromString(tenantIDHack))
)

func max(is ...int) int {
	max := is[0]
	for _, i := range is {
		if max < i {
			max = i
		}
	}
	return max
}

func (g *generator) templateFixup(t *arm.Template) ([]byte, error) {
	b, err := json.MarshalIndent(t, "", "    ")
	if err != nil {
		return nil, err
	}

	// :-(
	b = bytes.ReplaceAll(b, []byte(tenantIDHack), []byte("[subscription().tenantId]"))
	b = bytes.ReplaceAll(b, []byte(`"capacity": 1337`), []byte(`"capacity": "[parameters('ciCapacity')]"`))
	b = bytes.ReplaceAll(b, []byte(`"capacity": 1338`), []byte(`"capacity": "[parameters('rpVmssCapacity')]"`))
	if g.production {
		b = regexp.MustCompile(`(?m)"accessPolicies": \[[^]]*`+clusterAccessPolicyHack+`[^]]*\]`).ReplaceAll(b, []byte(`"accessPolicies": "[concat(variables('clusterKeyvaultAccessPolicies'), parameters('extraClusterKeyvaultAccessPolicies'))]"`))
		b = regexp.MustCompile(`(?m)"accessPolicies": \[[^]]*`+dbTokenAccessPolicyHack+`[^]]*\]`).ReplaceAll(b, []byte(`"accessPolicies": "[concat(variables('dbTokenKeyvaultAccessPolicies'), parameters('extraDBTokenKeyvaultAccessPolicies'))]"`))
		b = regexp.MustCompile(`(?m)"accessPolicies": \[[^]]*`+portalAccessPolicyHack+`[^]]*\]`).ReplaceAll(b, []byte(`"accessPolicies": "[concat(variables('portalKeyvaultAccessPolicies'), parameters('extraPortalKeyvaultAccessPolicies'))]"`))
		b = regexp.MustCompile(`(?m)"accessPolicies": \[[^]]*`+serviceAccessPolicyHack+`[^]]*\]`).ReplaceAll(b, []byte(`"accessPolicies": "[concat(variables('serviceKeyvaultAccessPolicies'), parameters('extraServiceKeyvaultAccessPolicies'))]"`))
		b = bytes.Replace(b, []byte(`"sourceAddressPrefixes": []`), []byte(`"sourceAddressPrefixes": "[parameters('rpNsgSourceAddressPrefixes')]"`), 1)
		b = bytes.Replace(b, []byte(`"isVirtualNetworkFilterEnabled": true`), []byte(`"isVirtualNetworkFilterEnabled": "[not(parameters('disableCosmosDBFirewall'))]"`), 1)
		b = bytes.Replace(b, []byte(`"virtualNetworkRules": []`), []byte(`"virtualNetworkRules": "[if(parameters('disableCosmosDBFirewall'), createArray(), createArray(createObject('id', resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet'))))]"`), 1)
		b = bytes.Replace(b, []byte(`"ipRules": []`), []byte(`"ipRules": "[if(parameters('disableCosmosDBFirewall'), createArray(), concat(parameters('ipRules'),createArray(createObject('ipAddressOrRange', '104.42.195.92'),createObject('ipAddressOrRange','40.76.54.131'),createObject('ipAddressOrRange','52.176.6.30'),createObject('ipAddressOrRange','52.169.50.45'),createObject('ipAddressOrRange','52.187.184.26'))))]"`), 1)

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
