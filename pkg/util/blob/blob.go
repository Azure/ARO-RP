package blob

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/url"
	"time"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

type BlobService interface {
	BlobExists(string, string) (bool, error)
	ReadBlob(string, string, *azstorage.GetBlobOptions) ([]byte, error)
	WriteBlob(string, string, []byte) error
	DeleteContainerIfExists(string) error
}

type blobService struct {
	c azstorage.BlobStorageClient
}

func NewBlobServiceForCluster(
	ctx context.Context,
	client storage.AccountsClient,
	cluster *api.OpenShiftCluster,
	p mgmtstorage.Permissions,
	r mgmtstorage.SignedResourceTypes,
) (BlobService, error) {
	resourceGroup := stringutils.LastTokenByte(cluster.Properties.ClusterProfile.ResourceGroupID, '/')

	t := time.Now().UTC().Truncate(time.Second)
	res, err := client.ListAccountSAS(ctx, resourceGroup, "cluster"+cluster.Properties.StorageSuffix, mgmtstorage.AccountSasParameters{
		Services:               mgmtstorage.B,
		ResourceTypes:          r,
		Permissions:            p,
		Protocols:              mgmtstorage.HTTPS,
		SharedAccessStartTime:  &date.Time{Time: t},
		SharedAccessExpiryTime: &date.Time{Time: t.Add(24 * time.Hour)},
	})
	if err != nil {
		return nil, err
	}

	v, err := url.ParseQuery(*res.AccountSasToken)
	if err != nil {
		return nil, err
	}

	c := azstorage.NewAccountSASClient("cluster"+cluster.Properties.StorageSuffix, v, azure.PublicCloud).GetBlobService()

	return &blobService{c: c}, nil
}

func (b *blobService) BlobExists(container string, reference string) (bool, error) {
	return b.c.GetContainerReference(container).GetBlobReference(reference).Exists()
}

func (b *blobService) ReadBlob(container string, reference string, options *azstorage.GetBlobOptions) ([]byte, error) {
	blob := b.c.GetContainerReference(container).GetBlobReference(reference)
	rc, err := blob.Get(options)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	content, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	return content, err
}

func (b *blobService) WriteBlob(container string, reference string, data []byte) error {
	blob := b.c.GetContainerReference(container).GetBlobReference(reference)
	err := blob.CreateBlockBlobFromReader(bytes.NewReader(data), nil)
	return err
}

func (b *blobService) DeleteContainerIfExists(container string) error {
	cont := b.c.GetContainerReference(container)
	_, err := cont.DeleteIfExists(&azstorage.DeleteContainerOptions{})
	return err
}

type fakeBlob struct {
	data map[string]map[string][]byte
}

func NewFakeBlobService() BlobService {
	d := make(map[string]map[string][]byte)
	return &fakeBlob{data: d}
}

func (b *fakeBlob) BlobExists(container string, reference string) (bool, error) {
	c, ext := b.data[container]
	if !ext {
		return false, nil
	}

	blob, ext := c[reference]
	if !ext {
		return false, nil
	}

	if blob == nil {
		return false, nil
	}

	return true, nil
}

func (b *fakeBlob) ReadBlob(container string, reference string, options *azstorage.GetBlobOptions) ([]byte, error) {
	c, ext := b.data[container]
	if !ext {
		return nil, errors.New("does not exist")
	}

	blob, ext := c[reference]
	if !ext || blob == nil {
		return nil, errors.New("does not exist")
	}

	return blob, nil
}

func (b *fakeBlob) WriteBlob(container string, reference string, data []byte) error {
	_, ext := b.data[container]
	if !ext {
		b.data[container] = make(map[string][]byte)
	}

	b.data[container][reference] = data
	return nil
}

func (b *fakeBlob) DeleteContainerIfExists(container string) error {
	b.data[container] = make(map[string][]byte)
	return nil
}
