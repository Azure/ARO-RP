package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const (
	rpVMSSPrefix      = "rp-vmss-"
	rpLBName          = "rp-lb"
	rpRuleName        = "rp-lbrule"
	portalRuleName    = "portal-lbrule"
	portalSSHRuleName = "portal-lbrule-ssh"
)

func (d *deployer) UpgradeRP(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	vmssName := rpVMSSPrefix + d.version
	scalesetVMs, err := d.rpWaitForReadiness(timeoutCtx, vmssName)
	if err != nil {
		// delete VMSS since VMSS instances are not healthy
		if *d.config.Configuration.VMSSCleanupEnabled {
			d.vmssCleaner.RemoveFailedNewScaleset(ctx, d.config.RPResourceGroupName, vmssName)
		}
		return err
	}

	err = d.rpWaitForLoadBalancerHealth(timeoutCtx, vmssName, scalesetVMs)
	if err != nil {
		if *d.config.Configuration.VMSSCleanupEnabled {
			d.vmssCleaner.RemoveFailedNewScaleset(ctx, d.config.RPResourceGroupName, vmssName)
		}
		return err
	}

	// Create a separate timeout for cleanup phase
	cleanupCtx, cleanupCancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cleanupCancel()
	return d.rpRemoveOldScalesets(cleanupCtx)
}

func (d *deployer) rpWaitForReadiness(ctx context.Context, vmssName string) ([]mgmtcompute.VirtualMachineScaleSetVM, error) {
	scalesetVMs, err := d.vmssvms.List(ctx, d.config.RPResourceGroupName, vmssName, "", "", "")
	if err != nil {
		return nil, err
	}

	d.log.Printf("waiting for %s instances to be healthy", vmssName)
	err = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		for _, vm := range scalesetVMs {
			if !d.isVMInstanceHealthy(ctx, d.config.RPResourceGroupName, vmssName, *vm.InstanceID) {
				return false, nil
			}
		}

		return true, nil
	}, ctx.Done())
	return scalesetVMs, err
}

func (d *deployer) rpWaitForLoadBalancerHealth(ctx context.Context, vmssName string, scalesetVMs []mgmtcompute.VirtualMachineScaleSetVM) error {
	if len(scalesetVMs) == 0 {
		return fmt.Errorf("VMSS %s has no instances", vmssName)
	}

	instanceIDs := make([]string, 0, len(scalesetVMs))
	for _, vm := range scalesetVMs {
		if vm.InstanceID == nil {
			return fmt.Errorf("VMSS instance has no instance ID")
		}
		instanceIDs = append(instanceIDs, *vm.InstanceID)
	}

	d.log.Printf("waiting for %s instances to be healthy on all RP load balancing rules", vmssName)
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		return d.rpLoadBalancerRulesHealthy(ctx, vmssName, instanceIDs)
	}, ctx.Done())
}

func (d *deployer) rpLoadBalancerRulesHealthy(ctx context.Context, vmssName string, instanceIDs []string) (bool, error) {
	for _, ruleName := range []string{rpRuleName, portalRuleName, portalSSHRuleName} {
		health, err := d.loadbalancingrules.HealthAndWait(ctx, d.config.RPResourceGroupName, rpLBName, ruleName, nil)
		if err != nil {
			return false, fmt.Errorf("getting health for RP load balancing rule %s: %w", ruleName, err)
		}

		for _, instanceID := range instanceIDs {
			backendFound := false
			instancePath := strings.ToLower(fmt.Sprintf("/virtualMachineScaleSets/%s/virtualMachines/%s/", vmssName, instanceID))
			for _, backend := range health.LoadBalancerBackendAddresses {
				if backend == nil || backend.NetworkInterfaceIPConfigurationID == nil || backend.NetworkInterfaceIPConfigurationID.ID == nil {
					continue
				}
				if strings.Contains(strings.ToLower(*backend.NetworkInterfaceIPConfigurationID.ID), instancePath) {
					backendFound = true
					if backend.State == nil || !strings.EqualFold(*backend.State, "Up") {
						return false, nil
					}
					break
				}
			}
			if !backendFound {
				return false, nil
			}
		}
	}

	return true, nil
}

func (d *deployer) rpRemoveOldScalesets(ctx context.Context) error {
	d.log.Print("removing old scalesets")
	scalesets, err := d.vmss.List(ctx, d.config.RPResourceGroupName)
	if err != nil {
		return err
	}

	for _, vmss := range scalesets {
		if *vmss.Name == rpVMSSPrefix+d.version {
			continue
		}

		err = d.rpRemoveOldScaleset(ctx, *vmss.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) rpRemoveOldScaleset(ctx context.Context, vmssName string) error {
	// Disable automatic repairs on the old VMSS to prevent race conditions during teardown
	err := d.disableAutomaticRepairsOnVMSS(ctx, d.config.RPResourceGroupName, vmssName)
	if err != nil {
		return err
	}

	scalesetVMs, err := d.vmssvms.List(ctx, d.config.RPResourceGroupName, vmssName, "", "", "")
	if err != nil {
		return err
	}

	d.log.Printf("stopping scaleset %s", vmssName)
	errors := make(chan error, len(scalesetVMs))
	for _, vm := range scalesetVMs {
		go func(id string) {
			errors <- d.runCommandWithRetry(ctx, d.config.RPResourceGroupName, vmssName, id, mgmtcompute.RunCommandInput{
				CommandID: pointerutils.ToPtr("RunShellScript"),
				Script:    &[]string{"systemctl stop aro-rp"},
			})
		}(*vm.InstanceID) // https://golang.org/doc/faq#closures_and_goroutines
	}

	d.log.Print("waiting for instances to stop")
	for range scalesetVMs {
		err := <-errors
		if err != nil {
			return err
		}
	}

	d.log.Printf("deleting scaleset %s", vmssName)
	return d.vmss.DeleteAndWait(ctx, d.config.RPResourceGroupName, vmssName)
}
