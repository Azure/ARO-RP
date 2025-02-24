package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	rpVMSSPrefix = "rp-vmss-"
)

func (d *deployer) UpgradeRP(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	err := d.rpWaitForReadiness(timeoutCtx, rpVMSSPrefix+d.version)
	if err != nil {
		// delete VMSS since VMSS instances are not healthy
		if *d.config.Configuration.VMSSCleanupEnabled {
			d.vmssCleaner.RemoveFailedNewScaleset(ctx, d.config.RPResourceGroupName, rpVMSSPrefix+d.version)
		}
		return err
	}

	return d.rpRemoveOldScalesets(ctx)
}

func (d *deployer) rpWaitForReadiness(ctx context.Context, vmssName string) error {
	scalesetVMs, err := d.vmssvms.List(ctx, d.config.RPResourceGroupName, vmssName, "", "", "")
	if err != nil {
		return err
	}

	d.log.Printf("waiting for %s instances to be healthy", vmssName)
	return wait.PollUntilContextCancel(ctx, 10*time.Second, true, func(ctx context.Context) (bool, error) {
		for _, vm := range scalesetVMs {
			if !d.isVMInstanceHealthy(ctx, d.config.RPResourceGroupName, vmssName, *vm.InstanceID) {
				return false, nil
			}
		}

		return true, nil
	})
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
	scalesetVMs, err := d.vmssvms.List(ctx, d.config.RPResourceGroupName, vmssName, "", "", "")
	if err != nil {
		return err
	}

	d.log.Printf("stopping scaleset %s", vmssName)
	errors := make(chan error, len(scalesetVMs))
	for _, vm := range scalesetVMs {
		go func(id string) {
			errors <- d.vmssvms.RunCommandAndWait(ctx, d.config.RPResourceGroupName, vmssName, id, mgmtcompute.RunCommandInput{
				CommandID: to.StringPtr("RunShellScript"),
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
