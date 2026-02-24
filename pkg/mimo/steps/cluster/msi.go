package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/msi-dataplane/pkg/dataplane"

	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/env"
	asazsecrets "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

func EnsureClusterMsiCertificate(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	oc := th.GetOpenshiftClusterDocument()

	if !oc.OpenShiftCluster.UsesWorkloadIdentity() {
		th.SetResultMessage("cluster does not use workload identity")
		return nil
	}

	taskEnv := th.Environment()

	msiKVURI := asazsecrets.URI(taskEnv, taskEnv.ClusterMsiKeyVaultName(), "")
	msiCredential, err := taskEnv.NewMSITokenCredential()
	if err != nil {
		return mimo.TerminalError(fmt.Errorf("failed to create MSI credential: %w", err))
	}

	kvStore, err := asazsecrets.NewClient(
		msiKVURI,
		msiCredential,
		taskEnv.Environment().AzureClientOptions(),
	)
	if err != nil {
		return mimo.TerminalError(fmt.Errorf("failed to create MSI KeyVault client: %w", err))
	}

	var msiDataplane dataplane.ClientFactory
	if taskEnv.FeatureIsSet(env.FeatureUseMockMsiRp) {
		clusterMsiResourceId, err := oc.OpenShiftCluster.ClusterMsiResourceId()
		if err != nil {
			return mimo.TerminalError(err)
		}
		msiDataplane = taskEnv.MockMSIResponses(clusterMsiResourceId)
	} else {
		msiDataplaneClientOptions, err := taskEnv.MsiDataplaneClientOptions(oc.CorrelationData)
		if err != nil {
			return mimo.TerminalError(fmt.Errorf("failed to get MSI dataplane client options: %w", err))
		}

		// MSI dataplane client receives tenant from the bearer challenge, so we can't limit the allowed tenants in the credential
		fpMSICred, err := taskEnv.FPNewClientCertificateCredential(taskEnv.TenantID(), []string{"*"})
		if err != nil {
			return mimo.TerminalError(fmt.Errorf("failed to create FP credential for MSI dataplane: %w", err))
		}

		msiDataplane = dataplane.NewClientFactory(fpMSICred, taskEnv.MsiRpEndpoint(), msiDataplaneClientOptions)
	}

	result, err := cluster.EnsureClusterMsiCertificateWithParams(ctx, oc.ID, oc.OpenShiftCluster, taskEnv.Now, kvStore, msiDataplane)
	if err != nil {
		return mimo.TransientError(fmt.Errorf("failed to ensure cluster MSI certificate: %w", err))
	}

	switch result {
	case cluster.MsiCertificateRefreshResultCreated:
		th.SetResultMessage("cluster MSI certificate created successfully")
	case cluster.MsiCertificateRefreshResultRenewed:
		th.SetResultMessage("cluster MSI certificate renewed successfully")
	case cluster.MsiCertificateRefreshResultUnchanged:
		th.SetResultMessage("cluster MSI certificate verified (no renewal needed)")
	}

	return nil
}
