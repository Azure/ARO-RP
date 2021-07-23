package node

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"
)

const (
	annotationCurrentConfig  = "machineconfiguration.openshift.io/currentConfig"
	annotationDesiredConfig  = "machineconfiguration.openshift.io/desiredConfig"
	annotationReason         = "machineconfiguration.openshift.io/reason"
	annotationState          = "machineconfiguration.openshift.io/state"
	annotationDrainStartTime = "aro.openshift.io/drainStartTime"
	stateDegraded            = "Degraded"
	stateWorking             = "Working"
	gracePeriod              = time.Hour
)
