package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/coreos/stream-metadata-go/stream"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const (
	AMD64Arch = "x86_64"
	ARM64Arch = "aarch64"

	resourceGroup = "images"
	accountName   = "openshiftimages"
)

func mirror(ctx context.Context, log *logrus.Entry, tokencred azcore.TokenCredential, version string, arch string) (string, error) {
	vhd, err := VHD(ctx, version, arch)
	if err != nil {
		return "", fmt.Errorf("VHD fetch error: %w", err)
	}

	name := stringutils.LastTokenByte(vhd, '/')
	log.Printf("Source VHD name: %s", name)
	log.Printf("Source VHD URL: %s", vhd)

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")

	accounts, err := armstorage.NewAccountsClient(subscriptionID, tokencred, nil)
	if err != nil {
		return "", fmt.Errorf("unexpected error creating accounts client: %w", err)
	}

	keys, err := accounts.ListKeys(ctx, resourceGroup, accountName, nil)
	if err != nil {
		return "", fmt.Errorf("unexpected error listing keys: %w", err)
	}

	// We need to use a shared key credential to create SAS tokens
	cred, err := azblob.NewSharedKeyCredential(accountName, *(keys.Keys)[0].Value)
	if err != nil {
		return "", fmt.Errorf("unexpected error creating shared key cred: %w", err)
	}

	// The mirroring process only needs to be performed in PublicCloud, partner
	// portal takes care of Govt
	serviceURL := fmt.Sprintf("https://%s.blob.%s", accountName, azure.PublicCloud.StorageEndpointSuffix)

	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		return "", fmt.Errorf("unexpected error creating shared key credential: %w", err)
	}

	b := client.ServiceClient().NewContainerClient("rhcos").NewPageBlobClient(name)

	p, err := b.GetProperties(ctx, nil)
	if err != nil && !bloberror.HasCode(err, bloberror.BlobNotFound) {
		return "", fmt.Errorf("unexpected error checking for blob: %w", err)
	}

	if p.Version == nil {
		u, err := b.StartCopyFromURL(ctx, vhd, nil)
		if err != nil {
			return "", fmt.Errorf("unexpected error starting copy: %w", err)
		}

		lastCopyProgress := ""
		copyStatus := *u.CopyStatus
		for copyStatus == blob.CopyStatusTypePending {
			time.Sleep(5 * time.Second)

			p, err = b.GetProperties(ctx, nil)
			if err != nil {
				return "", fmt.Errorf("unexpected error checking blob upload: %w", err)
			}
			copyStatus = *p.CopyStatus

			if lastCopyProgress != *p.CopyProgress {
				lastCopyProgress = *p.CopyProgress
				log.Printf("copy status: %s", *p.CopyProgress)
			}
		}

		log.Println("copy done")
	}

	sasuri, err := b.GetSASURL(sas.BlobPermissions{Read: true}, time.Now().AddDate(0, 0, 21), nil)
	if err != nil {
		return "", fmt.Errorf("unexpected error creating SAS URL: %w", err)
	}

	return sasuri, nil
}

// VHD fetches the URL of the public Azure blob containing the RHCOS image
func VHD(ctx context.Context, version string, rpmArchName string) (string, error) {
	rhcosURL := fmt.Sprintf("https://github.com/openshift/installer/raw/refs/heads/release-%s/data/data/coreos/rhcos.json", version)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rhcosURL, nil)
	if err != nil {
		return "", fmt.Errorf("unexpected error creating HTTP context: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error fetching %s: %w", rhcosURL, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unexpected error reading HTTP body: %w", err)
	}

	var st stream.Stream

	if err := json.Unmarshal(body, &st); err != nil {
		return "", fmt.Errorf("failed to parse coreOS stream metadata: %w", err)
	}

	streamArch, err := st.GetArchitecture(rpmArchName)
	if err != nil {
		return "", fmt.Errorf("unexpected error getting architecture '%s': %w", rpmArchName, err)
	}
	ext := streamArch.RHELCoreOSExtensions
	if ext == nil {
		return "", fmt.Errorf("%s: No azure build found", st.FormatPrefix(rpmArchName))
	}
	azd := ext.AzureDisk
	if azd == nil {
		return "", fmt.Errorf("%s: No azure build found", st.FormatPrefix(rpmArchName))
	}

	return azd.URL, nil
}

func main() {
	logger := logrus.New()
	log := logrus.NewEntry(logger)

	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <VERSION>\ne.g. %s 4.17", os.Args[0], os.Args[0])
	}

	version := os.Args[1]

	tokenCredential, err := azidentity.NewDeviceCodeCredential(&azidentity.DeviceCodeCredentialOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for _, i := range []string{AMD64Arch, ARM64Arch} {
		log.Infof("starting mirroring of %s, arch %s", version, i)
		url, err := mirror(context.Background(), log, tokenCredential, version, i)
		if err != nil {
			log.Fatalf("aborting: failed to mirror %s %s: %s", version, i, err.Error())
		}

		//fmt.Println so it goes to stdout so that the other logs can be piped
		//to a file if need be
		fmt.Println(url)
	}
}
