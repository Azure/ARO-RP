package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// LogLoadBalancers fetches the internal load balancer state and metrics and logs them.
func (m *manager) LogLoadBalancers(ctx context.Context) (interface{}, error) {
	if m.loadBalancers == nil || m.metrics == nil {
		return []interface{}{"load balancer or metrics client missing"}, nil
	}
	return []interface{}{}, m.logLoadBalancers(ctx)
}

func (m *manager) logLoadBalancers(ctx context.Context) error {

	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		return fmt.Errorf("InfraID is not set")
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
	// ResourceGroupID format: /subscriptions/{sub}/resourcegroups/{rg}
	// Split yields: ["", "subscriptions", "{sub}", "resourcegroups", "{rg}"]
	rgIDParts := strings.Split(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, "/")
	if len(rgIDParts) < 5 || rgIDParts[2] == "" || rgIDParts[4] == "" {
		return fmt.Errorf("invalid cluster resource group ID %q", m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)
	}
	subscriptionId := rgIDParts[2]
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	resp, err := m.loadBalancers.Get(ctx, resourceGroupName, lbName, nil)
	if err != nil {
		l.WithError(err).Errorf("failed to get load balancer")
		return err
	}
	lb := resp.LoadBalancer
	v, err := lb.MarshalJSON()
	if err != nil {
		l.WithError(err).Errorf("failed to marshal load balancer: %v", lb)
		return err
	}
	l.Infof("Load Balancer %s - %s", lbName, string(v))
	if lb.Properties != nil && lb.Properties.Probes != nil {
		for _, probe := range lb.Properties.Probes {
			probeName := "unnamed"
			if probe.Name != nil {
				probeName = *probe.Name
			}
			if probe.Properties != nil && probe.Properties.ProvisioningState != nil {
				l.Infof("Probe %s - provisioningstate=%s", probeName, *probe.Properties.ProvisioningState)
			}
		}
	}

	if lb.ID == nil {
		l.Errorf("load balancer %s has no ID; skipping metrics query", lbName)
		return nil
	}

	now := time.Now().UTC()
	startTime := now.Add(-time.Hour)
	result, err := m.metrics.QueryResources(
		ctx,
		subscriptionId,
		"Microsoft.Network/loadBalancers",
		[]string{"DipAvailability", "VipAvailability"},
		azmetrics.ResourceIDList{ResourceIDs: []string{*lb.ID}},
		&azmetrics.QueryResourcesOptions{
			Aggregation: pointerutils.ToPtr("Average"),
			StartTime:   pointerutils.ToPtr(startTime.Format(time.RFC3339)),
			EndTime:     pointerutils.ToPtr(now.Format(time.RFC3339)),
			Interval:    pointerutils.ToPtr("PT1M"),
		},
	)
	if err != nil {
		l.WithError(err).Errorf("failed to query health probe metrics for load balancer")
		return err
	}

	for _, resource := range result.Values {
		for _, metric := range resource.Values {
			metricName := ""
			if metric.Name != nil && metric.Name.Value != nil {
				metricName = *metric.Name.Value
			}
			for _, ts := range metric.TimeSeries {
				var dims []string
				for _, md := range ts.MetadataValues {
					if md.Name != nil && md.Name.Value != nil && md.Value != nil {
						dims = append(dims, fmt.Sprintf("%s=%s", *md.Name.Value, *md.Value))
					}
				}
				dimStr := strings.Join(dims, " ")
				// Coalesce consecutive identical values; log only edges where the value changes.
				var segStart *time.Time
				var segVal *float64
				for _, data := range ts.Data {
					if data.TimeStamp == nil || data.Average == nil {
						continue
					}
					if segVal == nil || *data.Average != *segVal {
						if segVal != nil {
							l.Infof("%s %s %s -> %s: %.0f%%", metricName, dimStr, segStart.Format(time.RFC3339), data.TimeStamp.Format(time.RFC3339), *segVal)
						}
						t := *data.TimeStamp
						segStart = &t
						segVal = data.Average
					}
				}
				if segVal != nil {
					l.Infof("%s %s %s: %.0f%%", metricName, dimStr, segStart.Format(time.RFC3339), *segVal)
				}
			}
		}
	}

	return nil
}
