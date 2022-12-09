package clusterdnschecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type clusterDNSChecker interface {
	Check(ctx context.Context) error
}

type checker struct {
	operatorcli operatorclient.Interface
}

func newClusterDNSChecker(operatorcli operatorclient.Interface) clusterDNSChecker {
	return &checker{
		operatorcli: operatorcli,
	}
}

func (r *checker) Check(ctx context.Context) error {
	dns, err := r.operatorcli.OperatorV1().DNSes().Get(ctx, "default", metav1.GetOptions{})
	if err != nil {
		return err
	}

	var upstreams []string
	for _, s := range dns.Spec.Servers {
		for _, z := range s.Zones {
			if z == "." {
				// If "." is set as a zone, bail out and warn about the
				// malformed config, as this will prevent CoreDNS from rolling
				// out
				return fmt.Errorf("malformed config: %q in zones", z)
			}
		}

		upstreams = append(upstreams, s.ForwardPlugin.Upstreams...)
	}

	if len(upstreams) > 0 {
		return fmt.Errorf("custom upstream DNS servers in use: %s", strings.Join(upstreams, ", "))
	}

	return nil
}
