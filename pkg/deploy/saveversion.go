package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

// SaveVersion for current location in shared storage account for environment
func (d *deployer) SaveVersion(ctx context.Context) error {
	d.log.Printf("saving RP version %s deployed in %s to storage account %s", d.version, d.config.Location, *d.config.Configuration.RPVersionStorageAccountName)

	d.log.Infof("ensuring static web content is enabled")
	_, err := d.blobsClient.ServiceClient().SetProperties(ctx, &service.SetPropertiesOptions{
		StaticWebsite: &service.StaticWebsite{Enabled: pointerutils.ToPtr(true)},
	})
	if err != nil {
		d.log.Errorf("failure to update static properties: %v", err)
		return err
	}

	d.log.Infof("uploading RP version")
	blobName := fmt.Sprintf("rpversion/%s", d.config.Location)
	_, err = d.blobsClient.UploadBuffer(ctx, "$web", blobName, []byte(d.version), nil)
	if err != nil {
		d.log.Errorf("failure to upload version information: %v", err)
		return err
	}

	return nil
}
