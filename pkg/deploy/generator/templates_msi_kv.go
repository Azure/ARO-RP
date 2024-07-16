package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func (g *generator) devMSIKeyvaultTemplate() *arm.Template {
	t := templateStanza()

	t.Resources = append(t.Resources,
		g.devMSIKeyvault(),
		g.devMSIKeyvaultRBAC(),
	)

	t.Parameters["rpServicePrincipalId"] = &arm.TemplateParameter{
		Type: "string",
	}

	return t
}
