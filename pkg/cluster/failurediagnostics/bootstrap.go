package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// LogBootstrapNodeState runs general diagnostic commands on the bootstrap VM
// and logs the output.
func (m *manager) LogBootstrapNodeState(ctx context.Context) (interface{}, error) {
	if m.virtualMachines == nil {
		return []interface{}{"vmclient missing"}, nil
	}
	return []interface{}{}, m.logBootstrapNodeState(ctx)
}

func (m *manager) logBootstrapNodeState(ctx context.Context) error {
	script := strings.Join([]string{
		"podman ps -a",
		"ss -tlnp",
		"systemctl list-units --state failed",
	}, "\n")

	return m.runBootstrapScript(ctx, "bootstrap node state", script)
}

// LogBootstrapMCS runs diagnostic commands on the bootstrap VM to check the
// status of the Machine Config Server (MCS) and logs the output.
func (m *manager) LogBootstrapMCS(ctx context.Context) (interface{}, error) {
	if m.virtualMachines == nil {
		return []interface{}{"vmclient missing"}, nil
	}
	return []interface{}{}, m.logBootstrapMCS(ctx)
}

func (m *manager) logBootstrapMCS(ctx context.Context) error {
	script := strings.Join([]string{
		"podman logs --tail 100 machine-config-server 2>&1 || true",
		"curl -v --insecure --head https://localhost:22623/config/master 2>&1 || true",
	}, "\n")

	return m.runBootstrapScript(ctx, "bootstrap MCS diagnostics", script)
}

// runBootstrapScript runs a shell script on the bootstrap VM via the Azure VM
// Run Command API and logs the output under the given label.
func (m *manager) runBootstrapScript(ctx context.Context, label, script string) error {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		return fmt.Errorf("InfraID is not set")
	}

	bootstrapVM := infraID + "-bootstrap"
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	result, err := m.virtualMachines.RunCommandAndWait(ctx, resourceGroupName, bootstrapVM, mgmtcompute.RunCommandInput{
		CommandID: pointerutils.ToPtr("RunShellScript"),
		Script:    &[]string{script},
	})
	if err != nil {
		m.log.WithError(err).Errorf("failed to run command on bootstrap VM %s", bootstrapVM)
		return err
	}

	if result.Value != nil {
		for _, status := range *result.Value {
			if status.Message != nil {
				code := ""
				if status.Code != nil {
					code = *status.Code
				}
				m.log.Infof("%s (%s):\n%s", label, code, *status.Message)
			}
		}
	}

	return nil
}
