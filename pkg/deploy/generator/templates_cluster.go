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
		"vnetAddressPrefix",
		"masterAddressPrefix",
		"workerAddressPrefix",
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "ci":
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
	)

	return t
}
