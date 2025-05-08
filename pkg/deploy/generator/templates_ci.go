package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func (g *generator) ciDevelopmentTemplate() *arm.Template {
	t := templateStanza()

	params := []string{
		"acrName",
		"acrLocationOverride",
		"acrResourceId",
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "acrLocationOverride":
			p.DefaultValue = "eastus"
		case "acrName":
			p.DefaultValue = "arosvcdev"
		case "acrResourceId":
			p.DefaultValue = "Microsoft.ContainerRegistry/registries"
		}
		t.Parameters[param] = p
	}
	t.Resources = append(t.Resources,
		g.ciACR(),
	)

	return t
}
