package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/openshift/installer/pkg/rhcos"

	_ "github.com/Azure/ARO-RP/pkg/install"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func run(ctx context.Context) error {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	resourceGroup := "images"
	accountName := "openshiftimages"

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	accounts := storage.NewAccountsClient(subscriptionID, authorizer)

	t := time.Now().UTC().Truncate(time.Second)

	res, err := accounts.ListAccountSAS(ctx, resourceGroup, accountName, mgmtstorage.AccountSasParameters{
		Services:               "b",
		ResourceTypes:          "co",
		Permissions:            "cr",
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

	blobService := azstorage.NewAccountSASClient(accountName, v, azure.PublicCloud).GetBlobService()

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
	log := utillog.GetLogger()

	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
