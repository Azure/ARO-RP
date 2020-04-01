package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"
	"time"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (d *deployer) Upgrade(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	err := d.waitForRPReadiness(timeoutCtx, vmssPrefix+d.version)
	if err != nil {
		return err
	}

	return d.removeOldScalesets(ctx)
}

func (d *deployer) waitForRPReadiness(ctx context.Context, vmssName string) error {
	scalesetVMs, err := d.vmssvms.List(ctx, d.config.ResourceGroupName, vmssName, "", "", "")
	if err != nil {
		return err
	}

	d.log.Printf("waiting for %s instances to be healthy", vmssName)
	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		for _, vm := range scalesetVMs {
			r, err := d.vmssvms.GetInstanceView(ctx, d.config.ResourceGroupName, vmssName, *vm.InstanceID)
			if err != nil || *r.VMHealth.Status.Code != "HealthState/healthy" {
				d.log.Printf("instance %s status %s", *vm.InstanceID, *r.VMHealth.Status.Code)
				return false, nil
			}
		}

		return true, nil
	}, ctx.Done())
}

func (d *deployer) removeOldScalesets(ctx context.Context) error {
	d.log.Print("removing old scalesets")
	scalesets, err := d.vmss.List(ctx, d.config.ResourceGroupName)
	if err != nil {
		return err
	}

	for _, vmss := range scalesets {
		if *vmss.Name == vmssPrefix+d.version {
			continue
		}

		d.log.Printf("stopping scaleset %s", *vmss.Name)
		err = d.runCommand(ctx, *vmss.Name, "systemctl stop aro-rp")
		if err != nil {
			return err
		}

		d.log.Printf("deleting scaleset %s", *vmss.Name)
		err = d.vmss.DeleteAndWait(ctx, d.config.ResourceGroupName, *vmss.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) runCommand(ctx context.Context, vmssName, command string) error {
	errorsCh := make(chan error)
	wgDone := make(chan bool)
	var wg sync.WaitGroup

	scalesetVMs, err := d.vmssvms.List(ctx, d.config.ResourceGroupName, vmssName, "", "", "")
	if err != nil {
		return err
	}

	for _, vm := range scalesetVMs {
		wg.Add(1)
		go func(id string) {
			err := d.vmssvms.RunCommandAndWait(ctx, d.config.ResourceGroupName, vmssName, id, mgmtcompute.RunCommandInput{
				CommandID: to.StringPtr("RunShellScript"),
				Script:    &[]string{command},
			})
			if err != nil {
				errorsCh <- err
			}
			wg.Done()
		}(*vm.InstanceID)

		go func() {
			wg.Wait()
			close(wgDone)
		}()

		select {
		case <-wgDone:
			break
		case err := <-errorsCh:
			close(errorsCh)
			return err
		}
	}
	return nil
}
