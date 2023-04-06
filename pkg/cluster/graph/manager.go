package graph

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/storage"
)

type Manager interface {
	Exists(ctx context.Context, resourceGroup, account string) (bool, error)
	Save(ctx context.Context, resourceGroup, account string, g Graph) error
	LoadPersisted(ctx context.Context, resourceGroup, account string) (PersistedGraph, error)
}

type manager struct {
	log *logrus.Entry

	aead    encryption.AEAD
	storage storage.Manager
}

func NewManager(log *logrus.Entry, aead encryption.AEAD, storage storage.Manager) Manager {
	return &manager{
		log: log,

		aead:    aead,
		storage: storage,
	}
}

func (m *manager) Exists(ctx context.Context, resourceGroup, account string) (bool, error) {
	m.log.Print("checking if graph exists")

	blobService, err := m.storage.BlobService(ctx, resourceGroup, account, mgmtstorage.Permissions("r"), mgmtstorage.SignedResourceTypesO)
	if err != nil {
		return false, err
	}

	aro := blobService.GetContainerReference("aro")
	return aro.GetBlobReference("graph").Exists()
}

// Load() should not be implemented: use LoadPersisted

func (m *manager) Save(ctx context.Context, resourceGroup, account string, g Graph) error {
	m.log.Print("save graph")

	blobService, err := m.storage.BlobService(ctx, resourceGroup, account, mgmtstorage.Permissions("cw"), mgmtstorage.SignedResourceTypesO)
	if err != nil {
		return err
	}

	bootstrap := g.Get(&bootstrap.Bootstrap{}).(*bootstrap.Bootstrap)
	bootstrapIgn := blobService.GetContainerReference("ignition").GetBlobReference("bootstrap.ign")
	err = bootstrapIgn.CreateBlockBlobFromReader(bytes.NewReader(bootstrap.File.Data), nil)
	if err != nil {
		return err
	}

	graph := blobService.GetContainerReference("aro").GetBlobReference("graph")
	b, err := json.MarshalIndent(g, "", "    ")
	if err != nil {
		return err
	}

	b, err = m.aead.Seal(b)
	if err != nil {
		return err
	}

	return graph.CreateBlockBlobFromReader(bytes.NewReader(b), nil)
}

func (m *manager) LoadPersisted(ctx context.Context, resourceGroup, account string) (PersistedGraph, error) {
	m.log.Print("load persisted graph")

	blobService, err := m.storage.BlobService(ctx, resourceGroup, account, mgmtstorage.Permissions("r"), mgmtstorage.SignedResourceTypesO)
	if err != nil {
		return nil, err
	}

	aro := blobService.GetContainerReference("aro")
	cluster := aro.GetBlobReference("graph")
	rc, err := cluster.Get(nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	b, err := io.ReadAll(rc)
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
