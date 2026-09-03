package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"maps"

	"github.com/google/uuid"

	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/installer/aksjob"
	"github.com/Azure/ARO-RP/pkg/installer/inputs"
)

func (m *manager) runAKSJobInstaller(ctx context.Context) error {
	// Persist the installer backend and execution ID before launching
	executionID := uuid.New().String()
	namespace := fmt.Sprintf("aro-install-%s", m.doc.OpenShiftCluster.Properties.InfraID)

	var err error
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.InstallerProfile = &api.InstallerProfile{
			Backend:     api.InstallerBackendAKSJob,
			ExecutionID: executionID,
			Namespace:   namespace,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to persist installer profile: %w", err)
	}

	// Get the OpenShift version
	version, err := m.openShiftVersionFromVersion(ctx)
	if err != nil {
		return err
	}

	// Prepare custom manifests
	customManifests := map[string]kruntime.Object{}
	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		workloadIdentityManifests, err := m.generateWorkloadIdentityResources()
		if err != nil {
			return err
		}
		maps.Copy(customManifests, workloadIdentityManifests)
	}

	if m.shouldDisableSamples() {
		customManifests["cluster-config-samples.yaml"] = bootstrapDisabledSamplesConfig()
	}

	// Build installer inputs
	builder := inputs.NewBuilder(m.env)
	installerInputs, err := builder.Build(m.doc, m.subscriptionDoc, version, customManifests)
	if err != nil {
		return fmt.Errorf("failed to build installer inputs: %w", err)
	}

	// Set namespace and execution ID
	installerInputs.Namespace = namespace
	installerInputs.ExecutionID = executionID

	// Get the RestConfig for the installer AKS cluster
	restConfig, err := m.env.LiveConfig().InstallerRestConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get installer AKS cluster config: %w", err)
	}

	// Create AKS Job installer manager
	aksJobManager, err := aksjob.New(m.log, restConfig)
	if err != nil {
		return fmt.Errorf("failed to create AKS job manager: %w", err)
	}

	// Run the installer
	m.log.Infof("starting AKS Job installer in namespace %s with execution ID %s", namespace, executionID)
	err = aksJobManager.Install(ctx, installerInputs)
	if err != nil {
		return fmt.Errorf("AKS job installer failed: %w", err)
	}

	m.log.Info("AKS Job installer completed successfully")
	return nil
}
