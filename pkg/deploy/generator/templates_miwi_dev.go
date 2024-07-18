package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func (g *generator) miwiDevSharedTemplate() *arm.Template {
	t := templateStanza()

	t.Resources = append(t.Resources,
		g.oicStorageAccount(),
		g.oicRoleAssignment(),
		g.devMSIKeyvault(),
		g.devMSIKeyvaultRBAC(),
	)

	t.Parameters = map[string]*arm.TemplateParameter{
		"rpServicePrincipalId": {
			Type: "string",
		},
		"oidcStorageAccountName": {
			Type: "string",
		},
	}

	return t
}
