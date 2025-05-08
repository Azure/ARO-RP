package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestUpdateFromIterator(t *testing.T) {
	for _, tt := range []struct {
		name           string
		docsInIterator []*api.GatewayDocument
		gateways       map[string]*api.Gateway
		wantGateways   map[string]*api.Gateway
	}{
		{
			name: "add to empty",
			docsInIterator: []*api.GatewayDocument{
				{ID: "notDeleting", Gateway: &api.Gateway{ID: "notDeleting"}},
			},
			gateways: map[string]*api.Gateway{},
			wantGateways: map[string]*api.Gateway{
				"notDeleting": {ID: "notDeleting"},
			},
		},
		{
			name: "do nothing",
			docsInIterator: []*api.GatewayDocument{
				{ID: "alreadyPresent", Gateway: &api.Gateway{ID: "alreadyPresent"}},
			},
			gateways: map[string]*api.Gateway{
				"alreadyPresent": {ID: "alreadyPresent"},
			},
			wantGateways: map[string]*api.Gateway{
				"alreadyPresent": {ID: "alreadyPresent"},
			},
		},
		{
			name: "add to not empty",
			docsInIterator: []*api.GatewayDocument{
				{ID: "notDeleting", Gateway: &api.Gateway{ID: "notDeleting"}},
			},
			gateways: map[string]*api.Gateway{
				"alreadyPresent": {ID: "alreadyPresent"},
			},
			wantGateways: map[string]*api.Gateway{
				"notDeleting":    {ID: "notDeleting"},
				"alreadyPresent": {ID: "alreadyPresent"},
			},
		},
		{
			name: "remove existing",
			docsInIterator: []*api.GatewayDocument{
				{ID: "alreadyPresent", Gateway: &api.Gateway{ID: "alreadyPresent"}},
				{ID: "to_delete", Gateway: &api.Gateway{ID: "to_delete", Deleting: true}},
			},
			gateways: map[string]*api.Gateway{
				"alreadyPresent": {ID: "alreadyPresent"},
				"to_delete":      {ID: "to_delete"},
			},
			wantGateways: map[string]*api.Gateway{
				"alreadyPresent": {ID: "alreadyPresent"},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ticker := time.NewTicker(1)
			ctx, cancel := context.WithCancel(context.TODO())

			gateway := gateway{
				gateways: tt.gateways,
			}

			fakeIterator := cosmosdb.NewFakeGatewayDocumentIterator(tt.docsInIterator, 0)

			go gateway.updateFromIterator(ctx, ticker, fakeIterator)
			time.Sleep(time.Second)
			cancel()

			if !reflect.DeepEqual(gateway.gateways, tt.wantGateways) {
				t.Error(cmp.Diff(gateway.gateways, tt.wantGateways))
			}
		})
	}
}
