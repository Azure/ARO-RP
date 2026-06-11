package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

type otelProfile string

const (
	otelProfileMaxLogs     otelProfile = operator.GenevaLoggingOTelProfileMaxLogs
	otelProfileReducedLogs otelProfile = operator.GenevaLoggingOTelProfileReducedLogs
	otelProfileMinimalLogs otelProfile = operator.GenevaLoggingOTelProfileMinimalLogs
)

type otelProfiles struct {
	master otelProfile
	worker otelProfile
}

func parseOTelProfile(profile string) (otelProfile, error) {
	switch profile {
	case operator.GenevaLoggingOTelProfileMaxLogs:
		return otelProfileMaxLogs, nil
	case operator.GenevaLoggingOTelProfileReducedLogs:
		return otelProfileReducedLogs, nil
	case operator.GenevaLoggingOTelProfileMinimalLogs:
		return otelProfileMinimalLogs, nil
	default:
		return "", fmt.Errorf(
			"unsupported geneva otel profile %q: expected %q, %q, or %q",
			profile,
			operator.GenevaLoggingOTelProfileMaxLogs,
			operator.GenevaLoggingOTelProfileReducedLogs,
			operator.GenevaLoggingOTelProfileMinimalLogs,
		)
	}
}

func getOTelProfiles(flags arov1alpha1.OperatorFlags) (otelProfiles, error) {
	globalProfileValue := flags.GetWithDefault(operator.GenevaLoggingOTelProfile, operator.GenevaLoggingOTelProfileMinimalLogs)
	masterProfileValue := flags.GetWithDefault(operator.GenevaLoggingOTelMasterProfile, globalProfileValue)
	workerProfileValue := flags.GetWithDefault(operator.GenevaLoggingOTelWorkerProfile, globalProfileValue)

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
