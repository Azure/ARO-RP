package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

func (m *manager) isIngressProfileAvailable() bool {
	// We try to acquire the IngressProfiles data at frontend best effort enrichment time only.
	// When we start deallocated VMs and wait for the API do become available again, we don't pick
	// the information up, even though it would be available.
	return len(m.doc.OpenShiftCluster.Properties.IngressProfiles) != 0
}

func (m *manager) ensureAROOperator(ctx context.Context) error {
	if !m.isIngressProfileAvailable() {
		// If the ingress profile is not available, ARO operator update/deploy will fail.
		m.log.Error("skip ensureAROOperator")
		return nil
	}

	err := m.aroOperatorDeployer.CreateOrUpdate(ctx)
	if err != nil {
		m.log.Errorf("cannot ensureAROOperator.CreateOrUpdate: %s", err.Error())
	}
	return err
}

func (m *manager) aroDeploymentReady(ctx context.Context) (bool, error) {
	if !m.isIngressProfileAvailable() {
		// If the ingress profile is not available, ARO operator update/deploy will fail.
		m.log.Error("skip aroDeploymentReady")
		return true, nil
	}
	return m.aroOperatorDeployer.IsReady(ctx)
}

func (m *manager) ensureAROOperatorRunningDesiredVersion(ctx context.Context) (bool, error) {
	if !m.isIngressProfileAvailable() {
		// If the ingress profile is not available, ARO operator update/deploy will fail.
		m.log.Error("skip ensureAROOperatorRunningDesiredVersion")
		return true, nil
	}
	ok, err := m.aroOperatorDeployer.IsRunningDesiredVersion(ctx)
	if !ok || err != nil {
		return false, err
	}
	return true, nil
}
