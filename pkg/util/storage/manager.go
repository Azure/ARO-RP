package storage

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	storagesdk "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armstorage"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azblob"
)

type Manager interface {
	BlobService(ctx context.Context, resourceGroup, account string, p storagesdk.Permissions, r storagesdk.SignedResourceTypes) (azblob.BlobsClient, error)
}

type manager struct {
	storageAccounts       armstorage.AccountsClient
	credential            azcore.TokenCredential
	usesWorkloadIdentity  bool
	storageEndpointSuffix string
	clientOptions         *arm.ClientOptions
}

func NewManager(subscriptionID, storageEndpointSuffix string, credential azcore.TokenCredential, usesWorkloadIdentity bool, options *arm.ClientOptions) (m Manager, err error) {
	var accountsClient armstorage.AccountsClient
	if !usesWorkloadIdentity {
		accountsClient, err = armstorage.NewAccountsClient(subscriptionID, credential, options)
		if err != nil {
			return nil, err
		}
	}
	return &manager{
		storageAccounts:       accountsClient,
		usesWorkloadIdentity:  usesWorkloadIdentity,
		credential:            credential,
		storageEndpointSuffix: storageEndpointSuffix,
		clientOptions:         options,
	}, nil
}

func getCorrectErrWhenTooManyRequests(err error) error {
	responseError, ok := err.(*azcore.ResponseError)
	if !ok {
		return err
	}
	if responseError.StatusCode != http.StatusTooManyRequests {
		return err
	}
	msg := "Requests are being throttled due to Azure Storage limits being exceeded. Please visit https://learn.microsoft.com/en-us/azure/openshift/troubleshoot#exceeding-azure-storage-limits for more details."
	cloudError := &api.CloudError{
		StatusCode: http.StatusTooManyRequests,
		CloudErrorBody: &api.CloudErrorBody{
			Code:    api.CloudErrorCodeThrottlingLimitExceeded,
			Message: "ThrottlingLimitExceeded",
			Details: []api.CloudErrorBody{
				{
					Message: msg,
				},
			},
		},
	}
	return cloudError
}

func (m *manager) BlobService(ctx context.Context, resourceGroup, account string, p storagesdk.Permissions, r storagesdk.SignedResourceTypes) (blobsClient azblob.BlobsClient, err error) {
	serviceURL := fmt.Sprintf("https://%s.blob.%s", account, m.storageEndpointSuffix)
	if m.usesWorkloadIdentity {
		return azblob.NewBlobsClientUsingEntra(serviceURL, m.credential, m.clientOptions)
	}

	t := time.Now().UTC().Truncate(time.Second)
	res, err := m.storageAccounts.ListAccountSAS(ctx, resourceGroup, account, storagesdk.AccountSasParameters{
		Services:               to.Ptr(storagesdk.ServicesB),
		ResourceTypes:          to.Ptr(r),
		Permissions:            to.Ptr(p),
		Protocols:              to.Ptr(storagesdk.HTTPProtocolHTTPS),
		SharedAccessStartTime:  &t,
		SharedAccessExpiryTime: to.Ptr(t.Add(24 * time.Hour)),
	}, nil)
	if err != nil {
		return nil, getCorrectErrWhenTooManyRequests(err)
	}

	_, err = url.ParseQuery(*res.AccountSasToken)
	if err != nil {
		return nil, err
	}

	sasURL := fmt.Sprintf("%s/?%s", serviceURL, *res.AccountSasToken)
	return azblob.NewBlobsClientUsingSAS(sasURL, m.clientOptions)
}
