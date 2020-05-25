package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
)

func (s *serialTool) dump(ctx context.Context, vmname string) error {
	vm, err := s.virtualMachines.Get(ctx, s.clusterResourceGroup, vmname, mgmtcompute.InstanceView)
	if err != nil {
		return err
	}

	u, err := url.Parse(*vm.InstanceView.BootDiagnostics.SerialConsoleLogBlobURI)
	if err != nil {
		return err
	}

	parts := strings.Split(u.Path, "/")
	if len(parts) != 3 {
		return fmt.Errorf("serialConsoleLogBlobURI has %d parts, expected 3", len(parts))
	}

	t := time.Now().UTC().Truncate(time.Second)
	res, err := s.accounts.ListAccountSAS(ctx, s.clusterResourceGroup, "cluster"+s.oc.Properties.StorageSuffix, mgmtstorage.AccountSasParameters{
		Services:               mgmtstorage.B,
		ResourceTypes:          mgmtstorage.SignedResourceTypesO,
		Permissions:            mgmtstorage.R,
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

	blobService := azstorage.NewAccountSASClient("cluster"+s.oc.Properties.StorageSuffix, v, azure.PublicCloud).GetBlobService()

	c := blobService.GetContainerReference(parts[1])

	b := c.GetBlobReference(parts[2])

	rc, err := b.Get(nil)
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(os.Stdout, rc)
	return err
}
