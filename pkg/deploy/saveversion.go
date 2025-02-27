package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azblob"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

// SaveVersion for current location in shared storage account for environment
func (d *deployer) SaveVersion(ctx context.Context, tokenCredential azcore.TokenCredential) error {
	d.log.Printf("saving RP version %s deployed in %s to storage account %s", d.version, d.config.Location, *d.config.Configuration.RPVersionStorageAccountName)

	d.log.Infof("instantiating blobs client using SAS token for ensure static web content is enabled")
	serviceUrl := fmt.Sprintf("https://%s.blob.%s", *d.config.Configuration.RPVersionStorageAccountName, d.env.Environment().StorageEndpointSuffix)
	blobsClient, err := azblob.NewBlobsClientUsingEntra(serviceUrl, tokenCredential, d.env.Environment().ArmClientOptions())
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

	d.log.Infof("instantiating blobs client using SAS token to upload content")
	containerUrl := fmt.Sprintf("https://%s.blob.%s/%s", *d.config.Configuration.RPVersionStorageAccountName, d.env.Environment().StorageEndpointSuffix, "$web")
	uploadBlobsClient, err := azblob.NewBlobsClientUsingEntra(containerUrl, tokenCredential, d.env.Environment().ArmClientOptions())
	if err != nil {
		d.log.Errorf("failure to instantiate blobs client using SAS: %v", err)
		return err
	}

	d.log.Infof("uploading RP version")
	blobName := fmt.Sprintf("rpversion/%s", d.config.Location)
	_, err = uploadBlobsClient.UploadBuffer(ctx, "$web", blobName, []byte(d.version), nil)
	if err != nil {
		d.log.Errorf("failure to upload version information: %v", err)
		return err
	}

	return nil
}
