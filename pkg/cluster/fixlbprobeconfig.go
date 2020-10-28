package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/pem"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) fixLBProbeConfig(ctx context.Context, resourceGroup, lbName string) error {
	mcsCertIsMalformed, err := m.mcsCertIsMalformed(ctx)
	if err != nil {
		return err
	}

	lb, err := m.loadbalancers.Get(ctx, resourceGroup, lbName, "")
	if err != nil {
		return err
	}

	if lb.LoadBalancerPropertiesFormat == nil || lb.LoadBalancerPropertiesFormat.Probes == nil {
		return nil
	}

	var changed bool

loop:
	for pix, probe := range *lb.LoadBalancerPropertiesFormat.Probes {
		protocol := mgmtnetwork.ProbeProtocolHTTPS
		var requestPath *string

		switch *probe.Name {
		case "api-internal-probe":
			requestPath = to.StringPtr("/readyz")
		case "sint-probe":
			if mcsCertIsMalformed {
				protocol = mgmtnetwork.ProbeProtocolTCP
			} else {
				requestPath = to.StringPtr("/healthz")
			}
		default:
			continue loop
		}

		if probe.ProbePropertiesFormat.Protocol != protocol {
			(*lb.LoadBalancerPropertiesFormat.Probes)[pix].ProbePropertiesFormat.Protocol = protocol
			changed = true
		}

		if !reflect.DeepEqual(probe.RequestPath, requestPath) {
			(*lb.LoadBalancerPropertiesFormat.Probes)[pix].RequestPath = requestPath
			changed = true
		}
	}

	if !changed {
		return nil
	}

	return m.loadbalancers.CreateOrUpdateAndWait(ctx, resourceGroup, lbName, lb)
}

func (m *manager) fixLBProbes(ctx context.Context) error {
	infraID := m.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	for _, lbName := range []string{
		infraID + "-public-lb",
		infraID + "-internal-lb",
	} {
		err := m.fixLBProbeConfig(ctx, resourceGroup, lbName)
		if err != nil {
			return err
		}
	}

	return nil
}

// mcsCertIsMalformed checks if the machine-config-server-tls certificate
// authority key identifier equals the subject key identifier, which is
// non-compliant and is rejected by Azure SLB.  This provisioning error was
// fixed in 4a7415a4 but clusters pre-dating the fix still exist.
func (m *manager) mcsCertIsMalformed(ctx context.Context) (bool, error) {
	s, err := m.kubernetescli.CoreV1().Secrets("openshift-machine-config-operator").Get(ctx, "machine-config-server-tls", metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	_, certs, err := pem.Parse(s.Data[v1.TLSCertKey])
	if err != nil {
		return false, err
	}

	if len(certs) == 0 {
		return false, fmt.Errorf("no certificate found")
	}

	return len(certs[0].AuthorityKeyId) > 0 &&
		bytes.Equal(certs[0].AuthorityKeyId, certs[0].SubjectKeyId), nil
}
