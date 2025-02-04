package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-09-01/storage"
	"github.com/Azure/go-autorest/autorest/date"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azblob"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

// SaveVersion for current location in shared storage account for environment
func (d *deployer) SaveVersion(ctx context.Context) error {
	d.log.Printf("saving RP version %s deployed in %s to storage account %s", d.version, d.config.Location, *d.config.Configuration.RPVersionStorageAccountName)
	t := time.Now().UTC().Truncate(time.Second)
	res, err := d.globalaccounts.ListAccountSAS(
		ctx, *d.config.Configuration.GlobalResourceGroupName, *d.config.Configuration.RPVersionStorageAccountName, mgmtstorage.AccountSasParameters{
			Services:               mgmtstorage.ServicesB,
			ResourceTypes:          mgmtstorage.SignedResourceTypesO + mgmtstorage.SignedResourceTypesS,
			Permissions:            mgmtstorage.PermissionsC + mgmtstorage.PermissionsW, // create and write
			Protocols:              mgmtstorage.HTTPProtocolHTTPS,
			SharedAccessStartTime:  &date.Time{Time: t},
			SharedAccessExpiryTime: &date.Time{Time: t.Add(24 * time.Hour)},
		})
	if err != nil {
		return err
	}

	d.log.Infof("instantiating blobs client using SAS token")
	sasUrl := fmt.Sprintf("https://%s.blob.%s/?%s", *d.config.Configuration.RPVersionStorageAccountName, d.env.Environment().StorageEndpointSuffix, *res.AccountSasToken)
	blobsClient, err := azblob.NewBlobsClientUsingSAS(sasUrl, d.env.Environment().ArmClientOptions())
	if err != nil {
		d.log.Errorf("failure to instantiate blobs client using SAS: %v", err)
		return err
	}

	d.log.Infof("ensuring static web content is enabled")
	_, err = blobsClient.ServiceClient().SetProperties(ctx, &service.SetPropertiesOptions{
		StaticWebsite: &service.StaticWebsite{Enabled: pointerutils.ToPtr(true)},
	})
	if err != nil {
		d.log.Errorf("failure to update static properties: %v", err)
		return err
	}

	d.log.Infof("uploading RP version")
	blobName := fmt.Sprintf("rpversion/%s", d.config.Location)
	_, err = blobsClient.UploadBuffer(ctx, "$web", blobName, []byte(d.version), nil)
	if err != nil {
		d.log.Errorf("failure to upload version information: %v", err)
		return err
	}

	return nil
}
