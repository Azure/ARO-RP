package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"testing"

	"github.com/coreos/go-semver/semver"

	"github.com/Azure/ARO-RP/pkg/api"
	v20220904 "github.com/Azure/ARO-RP/pkg/api/v20220904"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
)

func TestListInstallVersions(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	method := http.MethodGet
	ctx := context.Background()

	type test struct {
		name           string
		changeFeed     map[string]*api.OpenShiftVersion
		apiVersion     string
		wantStatusCode int
		wantResponse   v20220904.OpenShiftVersionList
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "return multiple versions",
			changeFeed: map[string]*api.OpenShiftVersion{
				"4.11.0": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.11.0",
						Enabled: true,
					},
				},
				"4.11.5": {
					Properties: api.OpenShiftVersionProperties{
						Version: "4.11.5",
						Enabled: true,
						Default: true,
					},
				},
			},
			apiVersion:     "2024-08-12-preview",
			wantStatusCode: http.StatusOK,
			wantResponse: v20220904.OpenShiftVersionList{
				OpenShiftVersions: []*v20220904.OpenShiftVersion{
					{
						Properties: v20220904.OpenShiftVersionProperties{
							Version: "4.11.0",
						},
						Type: api.OpenShiftVersionsType,
						Name: "4.11.0",
					},
					{
						Properties: v20220904.OpenShiftVersionProperties{
							Version: "4.11.5",
						},
						Type: api.OpenShiftVersionsType,
						Name: "4.11.5",
					},
				},
			},
		},
		{
			name:           "api does not exist",
			apiVersion:     "invalid",
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidResourceType: : The resource type '' could not be found in the namespace 'microsoft.redhatopenshift' for api version 'invalid'.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithSubscriptions().WithOpenShiftVersions()
			defer ti.done()

			frontend, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go frontend.Run(ctx, nil, nil)

			frontend.ocpVersionsMu.Lock()
			frontend.enabledOcpVersions = tt.changeFeed
			for key, doc := range tt.changeFeed {
				if doc.Properties.Enabled {
					frontend.defaultOcpVersion = key
				}
			}
			frontend.ocpVersionsMu.Unlock()

			resp, b, err := ti.request(method,
				fmt.Sprintf("https://server/subscriptions/%s/providers/Microsoft.RedHatOpenShift/locations/%s/openshiftversions?api-version=%s", mockSubID, ti.env.Location(), tt.apiVersion),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			// sort the response as the version order might be changed
			if b != nil && resp.StatusCode == http.StatusOK {
				var v v20220904.OpenShiftVersionList
				if err = json.Unmarshal(b, &v); err != nil {
					t.Error(err)
				}

				sort.Slice(v.OpenShiftVersions, func(i, j int) bool {
					return semver.New(v.OpenShiftVersions[i].Properties.Version).LessThan(*semver.New(v.OpenShiftVersions[j].Properties.Version))
				})

				b, err = json.Marshal(v)
				if err != nil {
					t.Error(err)
				}
			}

			// marshal the expected response into a []byte otherwise
			// it will compare zero values to omitempty json tags
			want, err := json.Marshal(tt.wantResponse)
			if err != nil {
				t.Error(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, want)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
