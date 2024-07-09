package storage

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/url"
	"time"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-09-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/date"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
)

type Manager interface {
	BlobService(ctx context.Context, resourceGroup, account string, p mgmtstorage.Permissions, r mgmtstorage.SignedResourceTypes) (*azstorage.BlobStorageClient, error)
}

type manager struct {
	env             env.Core
	storageAccounts storage.AccountsClient
}

func NewManager(env env.Core, subscriptionID string, authorizer autorest.Authorizer) Manager {
	return &manager{
		env:             env,
		storageAccounts: storage.NewAccountsClient(env.Environment(), subscriptionID, authorizer),
	}
}

func getCorrectErrWhenTooManyRequests(err error) error {
	detailedError, ok := err.(autorest.DetailedError)
	if !ok {
		return err
	}
	if detailedError.StatusCode != http.StatusTooManyRequests {
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

func (m *manager) BlobService(ctx context.Context, resourceGroup, account string, p mgmtstorage.Permissions, r mgmtstorage.SignedResourceTypes) (*azstorage.BlobStorageClient, error) {
	t := time.Now().UTC().Truncate(time.Second)
	res, err := m.storageAccounts.ListAccountSAS(ctx, resourceGroup, account, mgmtstorage.AccountSasParameters{
		Services:               mgmtstorage.ServicesB,
		ResourceTypes:          r,
		Permissions:            p,
		Protocols:              mgmtstorage.HTTPProtocolHTTPS,
		SharedAccessStartTime:  &date.Time{Time: t},
		SharedAccessExpiryTime: &date.Time{Time: t.Add(24 * time.Hour)},
	})
	if err != nil {
		return nil, getCorrectErrWhenTooManyRequests(err)
	}

	v, err := url.ParseQuery(*res.AccountSasToken)
	if err != nil {
		return nil, err
	}

	blobcli := azstorage.NewAccountSASClient(account, v, (*m.env.Environment()).Environment).GetBlobService()

	return &blobcli, nil
}
