package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/openshift/installer/pkg/types"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/rhcos"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func run(ctx context.Context, log *logrus.Entry) error {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	resourceGroup := "images"
	accountName := "openshiftimages"

	_env, err := env.NewCore(ctx, log)
	if err != nil {
		return err
	}

	authorizer, err := auth.NewAuthorizerFromCLIWithResource(_env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	accounts := storage.NewAccountsClient(_env.Environment(), subscriptionID, authorizer)

	keys, err := accounts.ListKeys(ctx, resourceGroup, accountName, "")
	if err != nil {
		return err
	}

	storagecli, err := azstorage.NewBasicClient(accountName, *(*keys.Keys)[0].Value)
	if err != nil {
		return err
	}

	blobService := storagecli.GetBlobService()

	c := blobService.GetContainerReference("rhcos")

	_, err = c.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	var vhd string
	if len(os.Args) == 2 {
		vhd = os.Args[1]
	} else {
		vhd, err = rhcos.VHD(ctx, types.ArchitectureAMD64)
		if err != nil {
			return err
		}
	}

	name := stringutils.LastTokenByte(vhd, '/')

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

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}
