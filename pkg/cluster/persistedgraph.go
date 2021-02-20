package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"reflect"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
)

// persistedGraph is a graph read from the cluster storage account.
// Unfortunately as the object schema changes over time, there are no guarantees
// that we can easily parse the objects in the graph, so we leave them as json
// RawMessages. You can expect get() to work in the context of cluster creation,
// but not necessarily subsequently.

type persistedGraph map[string]json.RawMessage

func (pg persistedGraph) get(is ...interface{}) error {
	for _, i := range is {
		d := json.NewDecoder(bytes.NewReader(pg[reflect.TypeOf(i).Elem().String()]))
		d.DisallowUnknownFields()

		err := d.Decode(i)
		if err != nil {
			return err
		}
	}

	return nil
}

// set is currently only used in unit test context.  If you want to use this in
// production, you will want to be very sure that you are not losing state that
// you may need later
func (pg persistedGraph) set(is ...interface{}) (err error) {
	for _, i := range is {
		pg[reflect.TypeOf(i).String()], err = json.Marshal(i)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) loadPersistedGraph(ctx context.Context) (persistedGraph, error) {
	m.log.Print("load persisted graph")

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

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

	b, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	b, err = m.aead.Open(b)
	if err != nil {
		return nil, err
	}

	var pg persistedGraph
	err = json.Unmarshal(b, &pg)
	if err != nil {
		return nil, err
	}

	return pg, nil
}

// savePersistedGraph could be implemented and used with care if needed, but
// currently we don't need it (and it's better that way)
