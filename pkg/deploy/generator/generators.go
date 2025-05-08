package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"os"

	"github.com/Azure/ARO-RP/pkg/util/arm"
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
		err := g.writeTemplate(g.rpManagedIdentityTemplate(), FileRPProductionManagedIdentity)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.rpPredeployTemplate(), FileRPProductionPredeploy)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.rpTemplate(), FileRPProduction)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.rpGlobalTemplate(), FileRPProductionGlobal)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.rpGlobalACRReplicationTemplate(), FileRPProductionGlobalACRReplication)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.rpGlobalSubscriptionTemplate(), FileRPProductionGlobalSubscription)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.rpSubscriptionTemplate(), FileRPProductionSubscription)
		if err != nil {
			return err
		}
		err = g.writeParameters(g.rpPredeployParameters(), FileRPProductionPredeployParameters)
		if err != nil {
			return err
		}
		err = g.writeParameters(g.rpParameters(), FileRPProductionParameters)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.gatewayManagedIdentityTemplate(), FileGatewayProductionManagedIdentity)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.gatewayPredeployTemplate(), FileGatewayProductionPredeploy)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.gatewayTemplate(), FileGatewayProduction)
		if err != nil {
			return err
		}
		err = g.writeParameters(g.gatewayPredeployParameters(), FileGatewayProductionPredeployParameters)
		if err != nil {
			return err
		}
		err = g.writeParameters(g.gatewayParameters(), FileGatewayProductionParameters)
		if err != nil {
			return err
		}
	} else {
		err := g.writeTemplate(g.devSharedTemplate(), fileEnvDevelopment)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.devDatabaseTemplate(), fileDatabaseDevelopment)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.rpPredeployTemplate(), fileRPDevelopmentPredeploy)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.rpTemplate(), fileRPDevelopment)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.clusterPredeploy(), FileClusterPredeploy)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.miwiDevSharedTemplate(), fileMiwiDevelopment)
		if err != nil {
			return err
		}
		err = g.writeTemplate(g.ciDevelopmentTemplate(), fileCIDevelopment)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *generator) writeTemplate(t *arm.Template, output string) error {
	b, err := g.templateFixup(t)
	if err != nil {
		return err
	}

	return os.WriteFile(output, b, 0666)
}

func (g *generator) writeParameters(p *arm.Parameters, output string) error {
	b, err := json.MarshalIndent(p, "", "    ")
	if err != nil {
		return err
	}
	b = append(b, byte('\n'))

	return os.WriteFile(output, b, 0666)
}
