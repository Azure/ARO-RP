package graph

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"io"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/storage"
)

const (
	GraphContainer    = "aro"
	GraphBlob         = "graph"
	IgnitionContainer = "ignition"
)

type Manager interface {
	Exists(ctx context.Context, resourceGroup, account string) (bool, error)
	LoadPersisted(ctx context.Context, resourceGroup, account string) (PersistedGraph, error)
}

type manager struct {
	log *logrus.Entry

	aead    encryption.AEAD
	storage storage.Manager
}

func NewManager(_ env.Interface, log *logrus.Entry, aead encryption.AEAD, storage storage.Manager) Manager {
	return &manager{
		log: log,

		aead:    aead,
		storage: storage,
	}
}

func (m *manager) Exists(ctx context.Context, resourceGroup, account string) (bool, error) {
	m.log.Print("checking if graph exists")

	blobService, err := m.storage.BlobService(ctx, resourceGroup, account, armstorage.Permissions("r"), armstorage.SignedResourceTypesO)
	if err != nil {
		return false, err
	}

	return blobService.BlobExists(ctx, GraphContainer, GraphBlob)
}

func (m *manager) LoadPersisted(ctx context.Context, resourceGroup, account string) (PersistedGraph, error) {
	return m.loadPersisted(ctx, resourceGroup, account)
}

func (m *manager) loadPersisted(ctx context.Context, resourceGroup, account string) (PersistedGraph, error) {
	m.log.Print("load persisted graph")

	blobService, err := m.storage.BlobService(ctx, resourceGroup, account, armstorage.Permissions("r"), armstorage.SignedResourceTypesO)
	if err != nil {
		return nil, err
	}

	rc, err := blobService.DownloadStream(ctx, GraphContainer, GraphBlob, nil)
	if err != nil {
		return nil, err
	}
	defer rc.Body.Close()

	b, err := io.ReadAll(rc.Body)
	if err != nil {
		return nil, err
	}

	b, err = m.aead.Open(b)
	if err != nil {
		return nil, err
	}

	var pg PersistedGraph
	err = json.Unmarshal(b, &pg)
	if err != nil {
		return nil, err
	}

	return pg, nil
}

// SavePersistedGraph could be implemented and used with care if needed, but
// currently we don't need it (and it's better that way)
