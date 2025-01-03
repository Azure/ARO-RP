package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
)

func TestGetOperations(t *testing.T) {
	method := http.MethodGet
	ctx := context.Background()

	// sort because ordering doesn't matter in
	sortFunc := func(a, b api.Operation) int {
		return strings.Compare(a.Name, b.Name)
	}

	allOperations := api.AllOperations
	slices.SortFunc(allOperations, sortFunc)

	type test struct {
		name           string
		apiVersion     string
		wantStatusCode int
		wantResponse   []api.Operation
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:           "return all operations",
			apiVersion:     "2022-09-04",
			wantStatusCode: http.StatusOK,
			wantResponse:   allOperations,
		},
		{
			name:           "api does not exist",
			apiVersion:     "invalid",
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidResourceType: : The resource type '' could not be found in the namespace 'microsoft.redhatopenshift' for api version 'invalid'.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t)
			defer ti.done()

			frontend, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go frontend.Run(ctx, nil, nil)

			resp, b, err := ti.request(method,
				fmt.Sprintf("https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=%s", tt.apiVersion),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Sort slices for consistent ordering
			if resp.StatusCode == http.StatusOK && b != nil {
				var actualOperations []api.Operation
				err = json.Unmarshal(b, &actualOperations)
				if err != nil {
					t.Error(err)
				}

				slices.SortFunc(actualOperations, sortFunc)
				b, err = json.Marshal(actualOperations)
				if err != nil {
					t.Error(err)
				}
			}

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
