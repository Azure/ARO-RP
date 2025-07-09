package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// LogVMSerialConsole fetches the serial console from VMs and logs them with
// the associated VM name.
func (m *manager) LogLoadBalancers(ctx context.Context) error {
	return m.logLoadBalancers(ctx, 50)
}

func (m *manager) logLoadBalancers(ctx context.Context, log_limit_kb int) error {
	if m.loadBalancers == nil {
		m.log.Info("skipping step")
		return nil
	}

	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	var lbName string
	switch m.doc.OpenShiftCluster.Properties.ArchitectureVersion {
	case api.ArchitectureVersionV1:
		lbName = infraID + "-internal-lb"
	case api.ArchitectureVersionV2:
		lbName = infraID + "-internal"
	default:
		return fmt.Errorf("unknown architecture version %d", m.doc.OpenShiftCluster.Properties.ArchitectureVersion)
	}

	l := m.log.WithField("lb", lbName)
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	subscriptionId := strings.Split(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, "/")[2]
	lb, err := m.loadBalancers.Get(ctx, resourceGroupName, lbName, "")
	if err != nil {
		l.WithError(err).Errorf("failed to get load balancer")
		return err
	}
	v, err := lb.MarshalJSON()
	if err != nil {
		l.WithError(err).Errorf("failed to marshal load balancer: %v", lb)
		return err
	}
	l.Infof("Load Balancer %s - %s", lbName, strings.ReplaceAll(string(v), "\"", "'"))
	if lb.Probes != nil {
		for _, probe := range *lb.Probes {
			probeName := "unnamed"
			if probe.Name != nil {
				probeName = *probe.Name
			}
			if probe.ProbePropertiesFormat != nil {
				l.Infof("Probe %s - provisioningstate=%s", probeName, probe.ProvisioningState)
			}
		}
	}

	metrics, err := m.metrics.QueryResources(
		ctx,
		subscriptionId,
		"Microsoft.Network",
		[]string{"NetworkSecurityGroup", "LoadBalancer", "PublicIPAddress"},
		azmetrics.ResourceIDList{ResourceIDs: []string{*lb.ID}},
		&azmetrics.QueryResourcesOptions{})
	if err != nil {
		l.WithError(err).Errorf("failed to query metrics for load balancer")
		return err
	}
	l.Infof("Load Balancer %s - metrics %v", lbName, metrics)
	for _, resource := range metrics.Values {
		for _, metric := range resource.Values {
			for _, timeseries := range metric.TimeSeries {
				for _, data := range timeseries.Data {
					l.Infof("Metric %s - %s: %f", *metric.Name.Value, data.TimeStamp.String(), *data.Total)
				}
			}
		}
	}

	return err
}
