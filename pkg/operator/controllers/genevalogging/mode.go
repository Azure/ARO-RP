// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

package genevalogging

import (
	"fmt"

	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

type otelProfile string

const (
	otelProfileMaxLogs     otelProfile = pkgoperator.GenevaLoggingOTelProfileMaxLogs
	otelProfileReducedLogs otelProfile = pkgoperator.GenevaLoggingOTelProfileReducedLogs
	otelProfileMinimalLogs otelProfile = pkgoperator.GenevaLoggingOTelProfileMinimalLogs
)

type otelProfiles struct {
	master otelProfile
	worker otelProfile
}

func parseOTelProfile(profile string) (otelProfile, error) {
	switch profile {
	case pkgoperator.GenevaLoggingOTelProfileMaxLogs:
		return otelProfileMaxLogs, nil
	case pkgoperator.GenevaLoggingOTelProfileReducedLogs:
		return otelProfileReducedLogs, nil
	case pkgoperator.GenevaLoggingOTelProfileMinimalLogs:
		return otelProfileMinimalLogs, nil
	default:
		return "", fmt.Errorf(
			"unsupported geneva otel profile %q: expected %q, %q, or %q",
			profile,
			pkgoperator.GenevaLoggingOTelProfileMaxLogs,
			pkgoperator.GenevaLoggingOTelProfileReducedLogs,
			pkgoperator.GenevaLoggingOTelProfileMinimalLogs,
		)
	}
}

func getOTelProfiles(flags arov1alpha1.OperatorFlags) (otelProfiles, error) {
	globalProfileValue := flags.GetWithDefault(pkgoperator.GenevaLoggingOTelProfile, pkgoperator.GenevaLoggingOTelProfileMinimalLogs)
	masterProfileValue := flags.GetWithDefault(pkgoperator.GenevaLoggingOTelMasterProfile, globalProfileValue)
	workerProfileValue := flags.GetWithDefault(pkgoperator.GenevaLoggingOTelWorkerProfile, globalProfileValue)

	masterProfile, err := parseOTelProfile(masterProfileValue)
	if err != nil {
		return otelProfiles{}, fmt.Errorf("master profile: %w", err)
	}

	workerProfile, err := parseOTelProfile(workerProfileValue)
	if err != nil {
		return otelProfiles{}, fmt.Errorf("worker profile: %w", err)
	}

	return otelProfiles{
		master: masterProfile,
		worker: workerProfile,
	}, nil
}
