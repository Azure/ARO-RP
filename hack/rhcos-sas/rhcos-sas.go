package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/openshift/installer/pkg/rhcos"

	_ "github.com/jim-minter/rp/pkg/install"
)

func run(ctx context.Context) error {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	resourceGroup := "images"
	accountName := "openshiftimages"

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	accounts := storage.NewAccountsClient(subscriptionID)
	accounts.Authorizer = authorizer

	keys, err := accounts.ListKeys(ctx, resourceGroup, accountName, "")
	if err != nil {
		return err
	}

	storageClient, err := azstorage.NewBasicClient(accountName, *(*keys.Keys)[0].Value)
	if err != nil {
		return err
	}

	blobService := storageClient.GetBlobService()

	c := blobService.GetContainerReference("rhcos")

	_, err = c.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	vhd, err := rhcos.VHD(ctx)
	if err != nil {
		return err
	}

	name := vhd[strings.LastIndexByte(vhd, '/')+1:]

	b := c.GetBlobReference(name)

	exists, err := b.Exists()
	if err != nil {
		return err
	}

	if !exists {
		err = b.Copy(vhd, nil)
		if err != nil {
			return err
		}
	}

	sasuri, err := c.GetSASURI(azstorage.ContainerSASOptions{
		ContainerSASPermissions: azstorage.ContainerSASPermissions{
			BlobServiceSASPermissions: azstorage.BlobServiceSASPermissions{
				Read: true,
			},
			List: true,
		},
		SASOptions: azstorage.SASOptions{
			Expiry: time.Now().AddDate(0, 0, 21),
		},
	})
	if err != nil {
		return err
	}

	fmt.Println(b.GetURL() + sasuri[strings.IndexByte(sasuri, '?'):])

	return nil
}

func main() {
	if err := run(context.Background()); err != nil {
		panic(err)
	}
}
