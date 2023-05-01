package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	"github.com/Azure/ARO-RP/pkg/util/steps"
)

func (m *manager) hiveStrategy() steps.Step {
	return steps.ListStep([]steps.Step{
		steps.Action(m.hiveCreateNamespace),
		steps.Action(m.runHiveInstaller),
		// Give Hive 60 minutes to install the cluster, since this includes
		// all of bootstrapping being complete
		steps.Condition(m.hiveClusterInstallationComplete, 60*time.Minute, true),
		steps.Condition(m.hiveClusterDeploymentReady, 5*time.Minute, true),
		steps.Action(m.generateKubeconfigs),
		steps.Action(m.hiveResetCorrelationData),
	})
}

func (m *manager) builtinStrategy() steps.Step {
	return steps.ListStep([]steps.Step{
		steps.Action(m.runIntegratedInstaller),
		steps.Action(m.generateKubeconfigs),
	})
}

func (m *manager) doHiveAdoptionIfConfigured() steps.Step {
	if m.adoptViaHive {
		return steps.ListStep([]steps.Step{
			steps.Action(m.hiveCreateNamespace),
			steps.Action(m.hiveEnsureResources),
			steps.Condition(m.hiveClusterDeploymentReady, 5*time.Minute, true),
			steps.Action(m.hiveResetCorrelationData),
		})
	} else {
		return steps.ListStep([]steps.Step{})
	}
}
