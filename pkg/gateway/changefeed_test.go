package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func TestUpdateFromIterator(t *testing.T) {

	for _, tt := range []struct {
		name                   string
		gatewaysAlreadyPresent []*api.Gateway
		docsInIterator         []*api.GatewayDocument
		wantAfterUpdate        map[string]*api.Gateway
	}{
		{
			name:            "add to empty",
			wantAfterUpdate: map[string]*api.Gateway{"notDeleting": {ID: "notDeleting"}},
			docsInIterator: []*api.GatewayDocument{
				{ID: "notDeleting", Gateway: &api.Gateway{ID: "notDeleting"}},
			},
		},
		{
			name: "do nothing",
			wantAfterUpdate: map[string]*api.Gateway{
				"alreadyPresent": {ID: "alreadyPresent"},
			},
			docsInIterator: []*api.GatewayDocument{
				{ID: "alreadyPresent", Gateway: &api.Gateway{ID: "alreadyPresent"}},
			},
			gatewaysAlreadyPresent: []*api.Gateway{
				{ID: "alreadyPresent"},
			},
		},
		{
			name: "add to not empty",
			wantAfterUpdate: map[string]*api.Gateway{
				"notDeleting":    {ID: "notDeleting"},
				"alreadyPresent": {ID: "alreadyPresent"},
			},
			docsInIterator: []*api.GatewayDocument{
				{ID: "notDeleting", Gateway: &api.Gateway{ID: "notDeleting"}},
			},
			gatewaysAlreadyPresent: []*api.Gateway{
				{ID: "alreadyPresent"},
			},
		},
		{
			name: "remove existing",
			wantAfterUpdate: map[string]*api.Gateway{
				"alreadyPresent": {ID: "alreadyPresent"},
			},
			docsInIterator: []*api.GatewayDocument{
				{ID: "alreadyPresent", Gateway: &api.Gateway{ID: "alreadyPresent"}},
				{ID: "to_delete", Gateway: &api.Gateway{ID: "to_delete", Deleting: true}},
			},
			gatewaysAlreadyPresent: []*api.Gateway{
				{ID: "alreadyPresent"},
				{ID: "to_delete"},
			},
		},
	} {
		gateway := gateway{
			gateways: make(map[string]*api.Gateway),
		}

		fakeIterator := cosmosdb.NewFakeGatewayDocumentIterator(tt.docsInIterator, 0)

		t.Run(tt.name, func(t *testing.T) {
			ticker := time.NewTicker(1)
			ctx, cancel := context.WithCancel(context.TODO())

			for _, v := range tt.gatewaysAlreadyPresent {
				gateway.gateways[v.ID] = v
			}

			go gateway.updateFromIterator(ctx, ticker, fakeIterator)
			time.Sleep(time.Second)
			cancel()

			//need to check both ways, to ensure that all that are in one
			//are in the other and vice versa
			for k, v := range gateway.gateways {
				if tt.wantAfterUpdate[k] == nil {
					t.Error(k)
				} else if v.ID != tt.wantAfterUpdate[k].ID {
					t.Error(k)
				}
			}
			for k, v := range tt.wantAfterUpdate {
				if gateway.gateways[k] == nil {
					t.Error(k)
				} else if v.ID != gateway.gateways[k].ID {
					t.Error(k)
				}
			}
		})

	}
}
