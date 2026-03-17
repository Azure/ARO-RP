package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// LogLoadBalancers fetches the internal load balancer state and Azure Monitor metrics and logs them.
func (m *manager) LogLoadBalancers(ctx context.Context) (interface{}, error) {
	if m.loadBalancers == nil || m.armMonitor == nil {
		return []interface{}{"load balancer or metrics client missing"}, nil
	}
	return []interface{}{}, m.logLoadBalancers(ctx)
}

func (m *manager) logLoadBalancers(ctx context.Context) error {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		return fmt.Errorf("infraID is not set")
	}

	lbName := infraID + "-internal"

	l := m.log.WithField("lb", lbName)
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	resp, err := m.loadBalancers.Get(ctx, resourceGroupName, lbName, nil)
	if err != nil {
		return fmt.Errorf("failed to get load balancer %s: %w", lbName, err)
	}
	lb := resp.LoadBalancer
	v, err := lb.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal load balancer %s: %w", lbName, err)
	}
	l.Infof("Load Balancer %s - %s", lbName, string(v))

	if lb.ID == nil {
		l.Errorf("load balancer %s has no ID; skipping metrics query", lbName)
		return nil
	}

	now := m.env.Now().UTC()
	startTime := now.Add(-time.Hour)
	timespan := fmt.Sprintf("%s/%s", startTime.Format(time.RFC3339), now.Format(time.RFC3339))

	result, err := m.armMonitor.List(
		ctx,
		*lb.ID,
		&armmonitor.MetricsClientListOptions{
			Metricnames: pointerutils.ToPtr("DipAvailability,VipAvailability"),
			Aggregation: pointerutils.ToPtr("average"),
			Timespan:    pointerutils.ToPtr(timespan),
			Interval:    pointerutils.ToPtr("PT1M"),
			// Split each metric by FrontendPort so that port 22623 (MCS)
			// and port 6443 (API) appear as separate time series.
			Filter: pointerutils.ToPtr("FrontendPort eq '*'"),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to query health probe metrics for load balancer %s: %w", lbName, err)
	}

	for _, metric := range result.Value {
		metricName := ""
		if metric.Name != nil && metric.Name.Value != nil {
			metricName = *metric.Name.Value
		}
		for _, ts := range metric.Timeseries {
			var dims []string
			for _, md := range ts.Metadatavalues {
				if md.Name != nil && md.Name.Value != nil && md.Value != nil {
					dims = append(dims, fmt.Sprintf("%s=%s", *md.Name.Value, *md.Value))
				}
			}
			dimStr := strings.Join(dims, " ")
			label := metricName
			if dimStr != "" {
				label += " " + dimStr
			}
			// Coalesce consecutive identical values; log only edges where the value changes.
			var segStart *time.Time
			// Round to nearest integer percent to avoid spurious edges from floating-point noise.
			var segVal *int64
			for _, data := range ts.Data {
				if data.TimeStamp == nil || data.Average == nil {
					continue
				}
				rounded := int64(math.Round(*data.Average))
				if segVal == nil || rounded != *segVal {
					if segVal != nil {
						l.Infof("%s %s -> %s: %d%%", label, segStart.Format(time.RFC3339), data.TimeStamp.Format(time.RFC3339), *segVal)
					}
					t := *data.TimeStamp
					segStart = &t
					segVal = &rounded
				}
			}
			if segVal != nil {
				l.Infof("%s %s: %d%%", label, segStart.Format(time.RFC3339), *segVal)
			}
		}
	}

	return nil
}
