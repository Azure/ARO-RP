package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"net/url"
	"time"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	vmssPrefix = "rp-vmss-"
)

func (d *deployer) Upgrade(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	err := d.waitForRPReadiness(timeoutCtx, vmssPrefix+d.version)
	if err != nil {
		return err
	}

	// maybe starting the new RP and stopping all the old ones in the same
	// minute is causing some of our alert noisiness.  Let's try a sleep and see
	// if that helps.
	d.log.Print("sleeping 5 minutes")
	time.Sleep(5 * time.Minute)

	err = d.removeOldScalesets(ctx)
	if err != nil {
		return err
	}

	// Must be last step so we can be sure there are no RPs at older versions still serving
	return d.saveRPVersion()
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

		err = d.removeOldScaleset(ctx, *vmss.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) removeOldScaleset(ctx context.Context, vmssName string) error {
	scalesetVMs, err := d.vmssvms.List(ctx, d.config.ResourceGroupName, vmssName, "", "", "")
	if err != nil {
		return err
	}

	d.log.Printf("stopping scaleset %s", vmssName)
	errors := make(chan error, len(scalesetVMs))
	for _, vm := range scalesetVMs {
		go func(id string) {
			errors <- d.vmssvms.RunCommandAndWait(ctx, d.config.ResourceGroupName, vmssName, id, mgmtcompute.RunCommandInput{
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
	return d.vmss.DeleteAndWait(ctx, d.config.ResourceGroupName, vmssName)
}

// saveRPVersion for current location in shared storage account for environment
func (d *deployer) saveRPVersion() error {
	d.log.Printf("saving rpVersion %s deployed in %s to storage account %s", d.version, d.config.Location, *d.config.Configuration.RPVersionStorageAccountName)
	t := time.Now().UTC().Truncate(time.Second)
	res, err := d.globalaccounts.ListAccountSAS(
		context.Background(), *d.config.Configuration.GlobalResourceGroupName, *d.config.Configuration.RPVersionStorageAccountName, mgmtstorage.AccountSasParameters{
			Services:               mgmtstorage.B,
			ResourceTypes:          mgmtstorage.SignedResourceTypesO,
			Permissions:            "cw", // create and write
			Protocols:              mgmtstorage.HTTPS,
			SharedAccessStartTime:  &date.Time{Time: t},
			SharedAccessExpiryTime: &date.Time{Time: t.Add(24 * time.Hour)},
		})
	if err != nil {
		return err
	}

	v, err := url.ParseQuery(*res.AccountSasToken)
	if err != nil {
		return err
	}

	blobClient := azstorage.NewAccountSASClient(
		*d.config.Configuration.RPVersionStorageAccountName, v, azure.PublicCloud).GetBlobService()

	containerRef := blobClient.GetContainerReference("rpversion")

	// save rpVersion deployed to current location
	blobRef := containerRef.GetBlobReference(d.config.Location)
	return blobRef.CreateBlockBlobFromReader(bytes.NewReader([]byte(d.version)), nil)
}
