package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func (g *generator) clusterPredeploy() *arm.Template {
	t := templateStanza()

	params := []string{
		"clusterName",
		"clusterServicePrincipalId",
		"fpServicePrincipalId",
		"ci",
		"routes",
		"vnetAddressPrefix",
		"masterAddressPrefix",
		"workerAddressPrefix",
		"kvName",
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "ci":
			p.Type = "bool"
			p.DefaultValue = false
		case "routes":
			p.Type = "array"
			p.DefaultValue = []interface{}{}
		}
		t.Parameters[param] = p
	}

	t.Resources = append(t.Resources,
		g.clusterVnet(),
		g.clusterRouteTable(),
		g.clusterMasterSubnet(),
		g.clusterWorkerSubnet(),
		g.diskEncryptionKeyVault(),
		g.diskEncryptionKey(),
		g.diskEncryptionSet(),
		g.diskEncryptionKeyVaultAccessPolicy(),
	)

	return t
}
