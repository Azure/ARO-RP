package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

type loggingMode string

const (
	loggingModeMDSD loggingMode = operator.GenevaLoggingModeMDSD
	loggingModeOTel loggingMode = operator.GenevaLoggingModeOTel
)

type otelProfile string

const (
	otelProfileHighLogLevel otelProfile = operator.GenevaLoggingOTelProfileHighLogLevel
	otelProfileReducedLogs  otelProfile = operator.GenevaLoggingOTelProfileReducedLogs
	otelProfileMinimalLogs  otelProfile = operator.GenevaLoggingOTelProfileMinimalLogs
)

// Backward-compatible aliases for existing tests and references.
const (
	otelProfileFull       otelProfile = otelProfileHighLogLevel
	otelProfileReduced    otelProfile = otelProfileReducedLogs
	otelProfileHighSignal otelProfile = otelProfileMinimalLogs
)

const (
	legacyOTelProfileFull       = "full"
	legacyOTelProfileReduced    = "reduced-noise"
	legacyOTelProfileHighSignal = "high-signal"
)

func getLoggingMode(flags arov1alpha1.OperatorFlags) (loggingMode, error) {
	mode := loggingMode(flags.GetWithDefault(operator.GenevaLoggingMode, operator.GenevaLoggingModeOTel))
	switch mode {
	case loggingModeMDSD, loggingModeOTel:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported geneva logging mode %q: expected %q or %q", mode, operator.GenevaLoggingModeMDSD, operator.GenevaLoggingModeOTel)
	}
}

type otelProfiles struct {
	master otelProfile
	worker otelProfile
}

func parseOTelProfile(profile string) (otelProfile, error) {
	switch profile {
	case operator.GenevaLoggingOTelProfileHighLogLevel, legacyOTelProfileFull:
		return otelProfileHighLogLevel, nil
	case operator.GenevaLoggingOTelProfileReducedLogs, legacyOTelProfileReduced:
		return otelProfileReducedLogs, nil
	case operator.GenevaLoggingOTelProfileMinimalLogs, legacyOTelProfileHighSignal:
		return otelProfileMinimalLogs, nil
	default:
		return "", fmt.Errorf(
			"unsupported geneva otel profile %q: expected %q, %q, or %q",
			profile,
			operator.GenevaLoggingOTelProfileHighLogLevel,
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

// getOTelProfile preserves legacy single-profile behavior by returning the
// global/default profile selection.
func getOTelProfile(flags arov1alpha1.OperatorFlags) (otelProfile, error) {
	profiles, err := getOTelProfiles(flags)
	if err != nil {
		return "", err
	}
	return profiles.master, nil
}
