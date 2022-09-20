package frontend

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
		docsInIterator []*api.OpenShiftVersionDocument
		versions       map[string]*api.OpenShiftVersion
		wantVersions   map[string]*api.OpenShiftVersion
	}{
		{
			name: "add to empty",
			docsInIterator: []*api.OpenShiftVersionDocument{
				{
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version: "4.2.0",
							Enabled: true,
						},
					},
				},
			},
			versions: map[string]*api.OpenShiftVersion{},
			wantVersions: map[string]*api.OpenShiftVersion{
				"4.2.0": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.2.0",
						Enabled: true,
					},
				},
			},
		},
		{
			name: "do nothing",
			docsInIterator: []*api.OpenShiftVersionDocument{
				{
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version: "4.5.6",
							Enabled: true,
						},
					},
				},
			},
			versions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
			},
			wantVersions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
			},
		},
		{
			name: "add to not empty",
			docsInIterator: []*api.OpenShiftVersionDocument{
				{
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version: "4.6.7",
							Enabled: true,
						},
					},
				},
			},
			versions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
			},
			wantVersions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
				"4.6.7": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.6.7",
						Enabled: true,
					},
				},
			},
		},
		{
			name: "remove existing",
			docsInIterator: []*api.OpenShiftVersionDocument{
				{
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version: "4.5.6",
							Enabled: true,
						},
					},
				},
				{
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version: "5.0.0",
							Enabled: true,
						},
						Deleting: true,
					},
				},
			},
			versions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
				"5.0.0": {
					Properties: api.OpenShiftVersionProperties{
						Version: "5.0.0",
						Enabled: true,
					},
				},
			},
			wantVersions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
			},
		},
		{
			name: "remove disabled versions",
			docsInIterator: []*api.OpenShiftVersionDocument{
				{
					OpenShiftVersion: &api.OpenShiftVersion{
						Properties: api.OpenShiftVersionProperties{
							Version: "4.5.6",
							Enabled: false,
						},
					},
				},
			},
			versions: map[string]*api.OpenShiftVersion{
				"4.5.6": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.5.6",
						Enabled: true,
					},
				},
			},
			wantVersions: map[string]*api.OpenShiftVersion{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ticker := time.NewTicker(1)
			ctx, cancel := context.WithCancel(context.TODO())

			frontend := frontend{
				enabledOcpVersions: tt.versions,
			}

			fakeIterator := cosmosdb.NewFakeOpenShiftVersionDocumentIterator(tt.docsInIterator, 0)

			go frontend.updateFromIterator(ctx, ticker, fakeIterator)
			time.Sleep(time.Second)
			cancel()

			if !reflect.DeepEqual(frontend.enabledOcpVersions, tt.wantVersions) {
				t.Error(cmp.Diff(frontend.enabledOcpVersions, tt.wantVersions))
			}
		})
	}
}
