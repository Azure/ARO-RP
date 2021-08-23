package vmsscleaner

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
)

type Interface interface {
	RemoveFailedScaleset(context.Context, string, string) bool
}

type cleaner struct {
	log  *logrus.Entry
	vmss compute.VirtualMachineScaleSetsClient
}

func New(log *logrus.Entry, vmss compute.VirtualMachineScaleSetsClient) Interface {
	return &cleaner{
		log:  log,
		vmss: vmss,
	}
}

func (c *cleaner) RemoveFailedScaleset(ctx context.Context, rgName, vmssName string) (deleted bool) {
	// Check if scaleset exists. If not, no need to delete it.
	vmss, err := c.vmss.Get(ctx, rgName, vmssName)
	if isNotFound(err) {
		return true
	}

	// If it is not in failed state, don't delete it.
	if *vmss.ProvisioningState != string(mgmtcompute.ProvisioningStateFailed) {
		return false
	}

	// If it is in failed state, try deleting so naming conflict doesn't occur during retry.
	c.log.Printf("deleting failed scaleset %s", *vmss.Name)
	err = c.vmss.DeleteAndWait(ctx, rgName, *vmss.Name)
	if err != nil {
		c.log.Warn(err)
		return false
	}
	return true
}

func isNotFound(err error) bool {
	if detailedErr, ok := err.(autorest.DetailedError); ok && detailedErr.StatusCode == http.StatusNotFound {
		return true
	}
	return false
}
