package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestUpdateGateway(t *testing.T) {

	for _, tt := range []struct {
		name                   string
		gatewaysAlreadyPresent []*api.Gateway
		doc                    *api.GatewayDocument
		wantValue              bool
		isDeleting             bool
	}{
		{
			name:      "add document",
			doc:       &api.GatewayDocument{ID: "notDeleting", Gateway: &api.Gateway{}},
			wantValue: true,
		},
		{
			name: "doesn't add document",
			doc: &api.GatewayDocument{ID: "deleting", Gateway: &api.Gateway{
				Deleting: true,
			}},
			wantValue: false,
		},
		{
			name:                   "remove document",
			gatewaysAlreadyPresent: []*api.Gateway{{ID: "toDelete"}},
			doc: &api.GatewayDocument{ID: "toDelete", Gateway: &api.Gateway{
				Deleting: true,
			}},
			wantValue: false,
		},
	} {

		gateway := gateway{
			gateways: make(map[string]*api.Gateway),
		}

		for _, v := range tt.gatewaysAlreadyPresent {
			gateway.gateways[v.ID] = v
		}

		t.Run(tt.name, func(t *testing.T) {
			gateway.updateGateways([]*api.GatewayDocument{tt.doc})

			if _, ok := gateway.gateways[tt.doc.ID]; ok != tt.wantValue {
				t.Error(tt.doc.ID)
			}
			if doc := gateway.gateways[tt.doc.ID]; tt.wantValue && doc == nil {
				t.Error(doc)
			}

		})

	}

}
