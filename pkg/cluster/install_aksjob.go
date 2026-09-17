package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"

	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/installer/aksjob"
	"github.com/Azure/ARO-RP/pkg/installer/inputs"
)

func (m *manager) runAKSJobInstaller(ctx context.Context) error {
	profile := m.doc.OpenShiftCluster.Properties.InstallerProfile
	if profile != nil && profile.Backend != api.InstallerBackendAKSJob {
		return fmt.Errorf("installer profile backend %q does not match selected backend %q", profile.Backend, api.InstallerBackendAKSJob)
	}

	if profile == nil || profile.ExecutionID == "" || profile.Namespace == "" {
		now := time.Now().UTC()
		executionID := uuid.New().String()
		profile = &api.InstallerProfile{
			Backend:     api.InstallerBackendAKSJob,
			ExecutionID: executionID,
			Namespace:   aksjob.Namespace,
			JobName:     aksjob.JobName(executionID),
			StartedAt:   &now,
		}

		var err error
		m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			doc.OpenShiftCluster.Properties.InstallerProfile = profile
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to persist installer profile: %w", err)
		}
	} else {
		m.log.Infof("resuming AKS Job installer in namespace %s with execution ID %s", profile.Namespace, profile.ExecutionID)
	}

	restConfig, err := m.env.LiveConfig().InstallerRestConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get installer AKS cluster config: %w", err)
	}

	aksJobManager, err := aksjob.New(m.log, restConfig)
	if err != nil {
		return fmt.Errorf("failed to create AKS job manager: %w", err)
	}

	if profile.CompletedAt != nil {
		if err := aksJobManager.Cleanup(ctx, profile.Namespace, profile.JobName); err != nil {
			m.log.WithError(err).Warn("failed to clean up completed AKS Job installer")
		}
		return nil
	}

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
	installerInputs.Namespace = profile.Namespace
	installerInputs.ExecutionID = profile.ExecutionID
	installerInputs.JobName = profile.JobName

	m.log.Infof("starting AKS Job installer in namespace %s with execution ID %s", profile.Namespace, profile.ExecutionID)
	err = aksJobManager.Install(ctx, installerInputs)
	if err != nil {
		if cleanupErr := aksJobManager.Cleanup(ctx, profile.Namespace, profile.JobName); cleanupErr != nil {
			m.log.WithError(cleanupErr).Error("failed to clean up unsuccessful AKS Job installer")
		}
		return err
	}

	completedAt := time.Now().UTC()
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		persisted := doc.OpenShiftCluster.Properties.InstallerProfile
		if persisted == nil || persisted.ExecutionID != profile.ExecutionID {
			return fmt.Errorf("installer profile changed while execution %s was running", profile.ExecutionID)
		}
		persisted.CompletedAt = &completedAt
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to persist completed installer profile: %w", err)
	}

	if err := aksJobManager.Cleanup(ctx, profile.Namespace, profile.JobName); err != nil {
		m.log.WithError(err).Warn("failed to clean up completed AKS Job installer")
	}

	m.log.Info("AKS Job installer completed successfully")
	return nil
}
