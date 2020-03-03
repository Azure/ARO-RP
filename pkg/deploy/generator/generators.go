package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"io/ioutil"

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

// Generator defines generator main interface
type Generator interface {
	Artifacts() error
}

type generator struct {
	production bool
}

func New(production bool) Generator {
	return &generator{
		production: production,
	}
}

func (g *generator) Artifacts() error {
	if g.production {
		err := g.writeTemplate(g.managedIdentityTemplate, FileRPProductionManagedIdentity)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.rpTemplate, FileRPProduction)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.preDeployTemplate, FileRPProductionPredeploy)
		if err != nil {
			return err
		}
		err = g.writeParameters(g.rpParameters, fileRPProductionParameters)
		if err != nil {
			return err
		}

		return g.writeParameters(g.rpPreDeployParameters, fileRPProductionPredeployParameters)
	} else {
		err := g.writeTemplate(g.rpTemplate, fileRPDevelopment)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.preDeployTemplate, fileRPDevelopmentPredeploy)
		if err != nil {
			return err
		}
		err = g.writeParameters(g.rpParameters, fileRPProductionParameters)
		if err != nil {
			return err
		}
		err = g.writeParameters(g.rpPreDeployParameters, fileRPProductionPredeployParameters)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.databaseTemplate, fileDatabaseDevelopment)
		if err != nil {
			return err
		}
		return g.writeTemplate(g.sharedDevelopmentEnvTemplate, fileEnvDevelopment)
	}

}

func (g *generator) writeTemplate(f func() *arm.Template, output string) error {
	b, err := g.templateFixup(f())
	if err != nil {
		return err
	}

	return ioutil.WriteFile(output, b, 0666)
}

func (g *generator) writeParameters(f func() *arm.Parameters, output string) error {
	b, err := json.MarshalIndent(f(), "", "    ")
	if err != nil {
		return err
	}
	b = append(b, byte('\n'))

	return ioutil.WriteFile(output, b, 0666)
}
