package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/cluster"
	"github.com/Azure/ARO-RP/pkg/cluster/graph"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/holmes"
	"github.com/Azure/ARO-RP/pkg/util/storage"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// generateDiagnosticsKubeconfig creates a short-lived (1 hour) kubeconfig for
// the system:aro-diagnostics identity. The kubeconfig is generated on each
// request using the cluster's CA from the persisted graph, so no long-lived
// credentials are stored in CosmosDB.
func (f *frontend) generateDiagnosticsKubeconfig(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument) ([]byte, error) {
	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return nil, err
	}

	credential, err := f.env.FPNewClientCertificateCredential(subscriptionDoc.Subscription.Properties.TenantID, nil)
	if err != nil {
		return nil, err
	}

	options := f.env.Environment().ArmClientOptions()
	storageManager, err := storage.NewManager(
		subscriptionDoc.ID,
		f.env.Environment().StorageEndpointSuffix,
		credential,
		doc.OpenShiftCluster.UsesWorkloadIdentity(),
		options,
	)
	if err != nil {
		return nil, err
	}

	clusterAead, err := encryption.NewMulti(ctx, f.env.ServiceKeyvault(), env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return nil, err
	}

	graphManager := graph.NewManager(f.env, log, clusterAead, storageManager)
	resourceGroup := stringutils.LastTokenByte(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + doc.OpenShiftCluster.Properties.StorageSuffix

	pg, err := graphManager.LoadPersisted(ctx, resourceGroup, account)
	if err != nil {
		return nil, err
	}

	kubeconfig, err := cluster.GenerateKubeconfig(pg, "system:aro-diagnostics", nil, time.Hour, true)
	if err != nil {
		return nil, err
	}

	// In development mode, the Hive cluster cannot resolve api-int.* private DNS
	// names, so we rewrite to the external api.* endpoint. In production, the
	// Hive cluster has proper network connectivity and should use api-int.* directly.
	if f.env.IsLocalDevelopmentMode() {
		kubeconfig, err = holmes.MakeExternalKubeconfig(kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	return kubeconfig, nil
}
