package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	v20240812preview "github.com/Azure/ARO-RP/pkg/api/v20240812preview"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
)

func TestGetInstallVersions(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	method := http.MethodGet
	ctx := context.Background()
	availableVersion := "4.14.5"
	changeFeed := map[string]*api.OpenShiftVersion{
		availableVersion: {
			Properties: api.OpenShiftVersionProperties{
				Version: availableVersion,
				Enabled: true,
				Default: true,
			},
			ID: "mockID",
		},
	}

	type test struct {
		name           string
		apiVersion     string
		version        string
		wantStatusCode int
		wantResponse   v20240812preview.OpenShiftVersion
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:           "return available version",
			apiVersion:     "2024-08-12-preview",
			version:        availableVersion,
			wantStatusCode: http.StatusOK,
			wantResponse: v20240812preview.OpenShiftVersion{
				Properties: v20240812preview.OpenShiftVersionProperties{
					Version: availableVersion,
				},
				Name: availableVersion,
				ID:   "mockID",
				Type: api.OpenShiftVersionsType,
			},
		},
		{
			name:           "api does not exist",
			apiVersion:     "invalid",
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidResourceType: : The resource type '' could not be found in the namespace 'microsoft.redhatopenshift' for api version 'invalid'.",
		},
		{
			name:           "openshift version not available",
			apiVersion:     "2024-08-12-preview",
			version:        "4.13.5",
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: ResourceNotFound: : The Resource openShiftVersion with version '4.13.5' was not found in the namespace 'microsoft.redhatopenshift' for api version '2024-08-12-preview'.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithSubscriptions().WithOpenShiftVersions()
			defer ti.done()

			frontend, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go frontend.Run(ctx, nil, nil)

			frontend.ocpVersionsMu.Lock()
			frontend.enabledOcpVersions = changeFeed
			for key, doc := range changeFeed {
				if doc.Properties.Enabled {
					frontend.defaultOcpVersion = key
				}
			}
			frontend.ocpVersionsMu.Unlock()

			resp, b, err := ti.request(method,
				fmt.Sprintf("https://server/subscriptions/%s/providers/Microsoft.RedHatOpenShift/locations/%s/openshiftversions/%s?api-version=%s", mockSubID, ti.env.Location(), tt.version, tt.apiVersion),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			// unmarshal and marshal the response body to match string content
			if b != nil && resp.StatusCode == http.StatusOK {
				var r v20240812preview.OpenShiftVersion
				if err = json.Unmarshal(b, &r); err != nil {
					t.Error(err)
				}

				b, err = json.Marshal(r)
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
