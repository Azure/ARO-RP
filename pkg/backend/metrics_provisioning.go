package backend

import (
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func (ocb *openShiftClusterBackend) emitProvisioningMetrics(doc *api.OpenShiftClusterDocument, provisioningState api.ProvisioningState) {
	if doc.CorrelationData == nil {
		return
	}

	duration := time.Since(doc.CorrelationData.RequestTime).Milliseconds()

	ocb.m.EmitGauge("backend.openshiftcluster.duration", duration, map[string]string{
		"oldProvisioningState": string(doc.OpenShiftCluster.Properties.ProvisioningState),
		"newProvisioningState": string(provisioningState),
	})

	ocb.m.EmitGauge("backend.openshiftcluster.count", 1, map[string]string{
		"oldProvisioningState": string(doc.OpenShiftCluster.Properties.ProvisioningState),
		"newProvisioningState": string(provisioningState),
	})
}
