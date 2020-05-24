package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

// disable removes load balancer and master nic configuration associated with
// SSH
func (s *sshTool) disable(ctx context.Context) error {
	lbName := s.infraID + "-internal-lb"

	lb, err := s.loadBalancers.Get(ctx, s.clusterResourceGroup, lbName, "")
	if err != nil {
		return err
	}

	for i := 0; i < 3; i++ {
		nicName := fmt.Sprintf("%s-master%d-nic", s.infraID, i)

		nic, err := s.interfaces.Get(ctx, s.clusterResourceGroup, nicName, "")
		if err != nil {
			return err
		}

		disableNIC(&nic, &lb, i)

		s.log.Printf("updating %s", nicName)
		err = s.interfaces.CreateOrUpdateAndWait(ctx, s.clusterResourceGroup, nicName, nic)
		if err != nil {
			return err
		}
	}

	disableLB(&lb)

	s.log.Printf("updating %s", lbName)
	return s.loadBalancers.CreateOrUpdateAndWait(ctx, s.clusterResourceGroup, lbName, lb)
}

// enable adds load balancer and master nic configuration associated with SSH
func (s *sshTool) enable(ctx context.Context) error {
	lbName := s.infraID + "-internal-lb"

	lb, err := s.loadBalancers.Get(ctx, s.clusterResourceGroup, lbName, "")
	if err != nil {
		return err
	}

	enableLB(&lb)

	s.log.Printf("updating %s", lbName)
	err = s.loadBalancers.CreateOrUpdateAndWait(ctx, s.clusterResourceGroup, lbName, lb)
	if err != nil {
		return err
	}

	for i := 0; i < 3; i++ {
		nicName := fmt.Sprintf("%s-master%d-nic", s.infraID, i)

		nic, err := s.interfaces.Get(ctx, s.clusterResourceGroup, nicName, "")
		if err != nil {
			return err
		}

		enableNIC(&nic, &lb, i)

		s.log.Printf("updating %s", nicName)
		err = s.interfaces.CreateOrUpdateAndWait(ctx, s.clusterResourceGroup, nicName, nic)
		if err != nil {
			return err
		}
	}

	return nil
}

func disableNIC(nic *mgmtnetwork.Interface, lb *mgmtnetwork.LoadBalancer, i int) {
	backendAddressPools := make([]mgmtnetwork.BackendAddressPool, 0, len(*(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools))
	for _, p := range *(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools {
		if !strings.EqualFold(*p.ID, fmt.Sprintf("%s/backendAddressPools/ssh-%d", *lb.ID, i)) {
			backendAddressPools = append(backendAddressPools, p)
		}
	}

	(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools = &backendAddressPools
}

func enableNIC(nic *mgmtnetwork.Interface, lb *mgmtnetwork.LoadBalancer, i int) {
	disableNIC(nic, lb, i)

	*(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools = append(*(*nic.IPConfigurations)[0].LoadBalancerBackendAddressPools, mgmtnetwork.BackendAddressPool{
		ID: to.StringPtr(fmt.Sprintf("%s/backendAddressPools/ssh-%d", *lb.ID, i)),
	})
}

func disableLB(lb *mgmtnetwork.LoadBalancer) {
	backendAddressPools := make([]mgmtnetwork.BackendAddressPool, 0, len(*lb.BackendAddressPools))
	for _, p := range *lb.BackendAddressPools {
		if !strings.HasPrefix(*p.Name, "ssh-") {
			backendAddressPools = append(backendAddressPools, p)
		}
	}
	*lb.BackendAddressPools = backendAddressPools

	loadBalancingRules := make([]mgmtnetwork.LoadBalancingRule, 0, len(*lb.LoadBalancingRules))
	for _, c := range *lb.LoadBalancingRules {
		if !strings.HasPrefix(*c.Name, "ssh-") {
			loadBalancingRules = append(loadBalancingRules, c)
		}
	}
	*lb.LoadBalancingRules = loadBalancingRules

	probes := make([]mgmtnetwork.Probe, 0, len(*lb.Probes))
	for _, p := range *lb.Probes {
		if *p.Name != "ssh" {
			probes = append(probes, p)
		}
	}
	*lb.Probes = probes
}

func enableLB(lb *mgmtnetwork.LoadBalancer) {
	disableLB(lb)

	for i := int32(0); i < 3; i++ {
		*lb.BackendAddressPools = append(*lb.BackendAddressPools, mgmtnetwork.BackendAddressPool{
			Name: to.StringPtr(fmt.Sprintf("ssh-%d", i)),
		})

		*lb.LoadBalancingRules = append(*lb.LoadBalancingRules, mgmtnetwork.LoadBalancingRule{
			LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
				FrontendIPConfiguration: &mgmtnetwork.SubResource{
					ID: (*lb.FrontendIPConfigurations)[0].ID,
				},
				BackendAddressPool: &mgmtnetwork.SubResource{
					ID: to.StringPtr(fmt.Sprintf("%s/backendAddressPools/ssh-%d", *lb.ID, i)),
				},
				Probe: &mgmtnetwork.SubResource{
					ID: to.StringPtr(*lb.ID + "/probes/ssh"),
				},
				Protocol:             mgmtnetwork.TransportProtocolTCP,
				LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
				FrontendPort:         to.Int32Ptr(2200 + i),
				BackendPort:          to.Int32Ptr(22),
				IdleTimeoutInMinutes: to.Int32Ptr(30),
				DisableOutboundSnat:  to.BoolPtr(true),
			},
			Name: to.StringPtr(fmt.Sprintf("ssh-%d", i)),
		})
	}

	*lb.Probes = append(*lb.Probes, mgmtnetwork.Probe{
		ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
			Protocol:          mgmtnetwork.ProbeProtocolTCP,
			Port:              to.Int32Ptr(22),
			IntervalInSeconds: to.Int32Ptr(10),
			NumberOfProbes:    to.Int32Ptr(3),
		},
		Name: to.StringPtr("ssh"),
	})
}
