package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const (
	gatewayVMSSPrefix = "gateway-vmss-"
)

func (d *deployer) UpgradeGateway(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 40*time.Minute)
	defer cancel()
	err := d.gatewayWaitForReadiness(timeoutCtx, gatewayVMSSPrefix+d.version)
	if err != nil {
		// delete VMSS since VMSS instances are not healthy
		if *d.config.Configuration.VMSSCleanupEnabled {
			d.vmssCleaner.RemoveFailedNewScaleset(ctx, d.config.GatewayResourceGroupName, gatewayVMSSPrefix+d.version)
		}
		return err
	}

	return d.gatewayRemoveOldScalesets(ctx)
}

func (d *deployer) gatewayWaitForReadiness(ctx context.Context, vmssName string) error {
	scalesetVMs, err := d.vmssvms.List(ctx, d.config.GatewayResourceGroupName, vmssName, "", "", "")
	if err != nil {
		return err
	}

	d.log.Printf("waiting for %s instances to be healthy", vmssName)
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		for _, vm := range scalesetVMs {
			if !d.isVMInstanceHealthy(ctx, d.config.GatewayResourceGroupName, vmssName, *vm.InstanceID) {
				return false, nil
			}
		}

		return true, nil
	}, ctx.Done())
}

func (d *deployer) gatewayRemoveOldScalesets(ctx context.Context) error {
	d.log.Print("removing old scalesets")
	scalesets, err := d.vmss.List(ctx, d.config.GatewayResourceGroupName)
	if err != nil {
		return err
	}

	for _, vmss := range scalesets {
		if *vmss.Name == gatewayVMSSPrefix+d.version {
			continue
		}

		err = d.gatewayRemoveOldScaleset(ctx, *vmss.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) gatewayRemoveOldScaleset(ctx context.Context, vmssName string) error {
	scalesetVMs, err := d.vmssvms.List(ctx, d.config.GatewayResourceGroupName, vmssName, "", "", "")
	if err != nil {
		return err
	}

	d.log.Printf("stopping scaleset %s", vmssName)
	errors := make(chan error, len(scalesetVMs))
	for _, vm := range scalesetVMs {
		if d.isVMInstanceHealthy(ctx, d.config.GatewayResourceGroupName, vmssName, *vm.InstanceID) {
			d.log.Printf("stopping gateway service on %s", *vm.Name)
			go func(id string) {
				errors <- d.vmssvms.RunCommandAndWait(ctx, d.config.GatewayResourceGroupName, vmssName, id, mgmtcompute.RunCommandInput{
					CommandID: pointerutils.ToPtr("RunShellScript"),
					Script:    &[]string{"systemctl stop aro-gateway"},
				})
			}(*vm.InstanceID) // https://golang.org/doc/faq#closures_and_goroutines
		}
	}

	d.log.Print("waiting for instances to stop")
	for range scalesetVMs {
		select {
		case err := <-errors:
			if err != nil {
				return err
			}
		case <-time.After(10 * time.Second):
			continue
		}
	}

	d.log.Printf("deleting scaleset %s", vmssName)
	return d.vmss.DeleteAndWait(ctx, d.config.GatewayResourceGroupName, vmssName)
}
