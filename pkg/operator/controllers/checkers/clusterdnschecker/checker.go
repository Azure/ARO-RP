package clusterdnschecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "github.com/openshift/api/operator/v1"
)

type clusterDNSChecker interface {
	Check(ctx context.Context) (result, error)
}

type result struct {
	success bool
	message string
}

type checker struct {
	client client.Client
}

func newClusterDNSChecker(client client.Client) *checker {
	return &checker{
		client: client,
	}
}

func (r *checker) Check(ctx context.Context) (result, error) {
	dns := &operatorv1.DNS{}
	err := r.client.Get(ctx, types.NamespacedName{Name: "default"}, dns)
	if err != nil {
		return result{}, err
	}

	var upstreams []string
	for _, s := range dns.Spec.Servers {
		for _, z := range s.Zones {
			if z == "." {
				// If "." is set as a zone, bail out and warn about the
				// malformed config, as this will prevent CoreDNS from rolling
				// out.
				return result{false, fmt.Sprintf("malformed config: %q in zones", z)}, nil
			}
		}

		upstreams = append(upstreams, s.ForwardPlugin.Upstreams...)
	}

	if len(upstreams) > 0 {
		// Custom DNS servers are a supported setup as per our docs
		// https://learn.microsoft.com/en-us/azure/openshift/dns-forwarding
		// We still report them here for awareness.
		return result{true, fmt.Sprintf("custom upstream DNS servers in use: %s", strings.Join(upstreams, ", "))}, nil
	}

	return result{true, "no in-cluster upstream DNS servers"}, nil
}
