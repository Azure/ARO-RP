package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"net/url"
	"time"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/date"
)

// SaveVersion for current location in shared storage account for environment
func (d *deployer) SaveVersion(ctx context.Context) error {
	d.log.Printf("saving rpVersion %s deployed in %s to storage account %s", d.version, d.config.Location, *d.config.Configuration.RPVersionStorageAccountName)
	t := time.Now().UTC().Truncate(time.Second)
	res, err := d.globalaccounts.ListAccountSAS(
		ctx, *d.config.Configuration.GlobalResourceGroupName, *d.config.Configuration.RPVersionStorageAccountName, mgmtstorage.AccountSasParameters{
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
		*d.config.Configuration.RPVersionStorageAccountName, v, (*d.env.Environment()).Environment).GetBlobService()

	containerRef := blobClient.GetContainerReference("rpversion")

	// save rpVersion deployed to current location
	blobRef := containerRef.GetBlobReference(d.config.Location)
	return blobRef.CreateBlockBlobFromReader(bytes.NewReader([]byte(d.version)), nil)
}
