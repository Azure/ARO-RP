package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
)

// graph is used to generate and persist the graph as a one-off.  For subsequent
// uses, use persistedGraph.

type graph map[string]asset.Asset

func (g graph) get(a asset.Asset) asset.Asset {
	return g[reflect.TypeOf(a).String()]
}

func (g graph) set(as ...asset.Asset) {
	for _, a := range as {
		g[reflect.TypeOf(a).String()] = a
	}
}

func (g graph) resolve(a asset.Asset) error {
	if g.get(a) != nil {
		return nil
	}

	for _, dep := range a.Dependencies() {
		err := g.resolve(dep)
		if err != nil {
			return err
		}
	}

	parents := asset.Parents{}
	for _, v := range g {
		parents[reflect.TypeOf(v)] = v
	}

	err := a.Generate(parents)
	if err != nil {
		return err
	}

	g.set(a)

	return nil
}

func (m *manager) graphExists(ctx context.Context) (bool, error) {
	m.log.Print("checking if graph exists")

	blobService, err := m.getBlobService(ctx, mgmtstorage.Permissions("r"), mgmtstorage.SignedResourceTypesO)
	if err != nil {
		return false, err
	}

	aro := blobService.GetContainerReference("aro")
	return aro.GetBlobReference("graph").Exists()
}

// loadGraph() should not be implemented: use loadPersistedGraph

func (m *manager) saveGraph(ctx context.Context, g graph) error {
	m.log.Print("save graph")

	blobService, err := m.getBlobService(ctx, mgmtstorage.Permissions("cw"), mgmtstorage.SignedResourceTypesO)
	if err != nil {
		return err
	}

	bootstrap := g.get(&bootstrap.Bootstrap{}).(*bootstrap.Bootstrap)
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
