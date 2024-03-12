package clusterdnschecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	operatorv1 "github.com/openshift/api/operator/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/ARO-RP/pkg/operator/metrics"
)

type clusterDNSChecker interface {
	Check(ctx context.Context) error
}

type checker struct {
	client        client.Client
	metricsClient metrics.Client
}

func newClusterDNSChecker(client client.Client, metricsClient metrics.Client) *checker {
	return &checker{
		client:        client,
		metricsClient: metricsClient,
	}
}

func (r *checker) Check(ctx context.Context) error {
	dns := &operatorv1.DNS{}
	err := r.client.Get(ctx, types.NamespacedName{Name: "default"}, dns)
	if err != nil {
		r.metricsClient.UpdateDnsConfigurationValid(false)
		return err
	}

	var upstreams []string
	for _, s := range dns.Spec.Servers {
		for _, z := range s.Zones {
			if z == "." {
				// If "." is set as a zone, bail out and warn about the
				// malformed config, as this will prevent CoreDNS from rolling
				// out
				r.metricsClient.UpdateDnsConfigurationValid(false)
				return fmt.Errorf("malformed config: %q in zones", z)
			}
		}

		upstreams = append(upstreams, s.ForwardPlugin.Upstreams...)
	}

	if len(upstreams) > 0 {
		r.metricsClient.UpdateDnsConfigurationValid(false)
		return fmt.Errorf("custom upstream DNS servers in use: %s", strings.Join(upstreams, ", "))
	}

	r.metricsClient.UpdateDnsConfigurationValid(true)
	return nil
}
