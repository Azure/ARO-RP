package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

// transportFunc adapts a function to policy.Transporter so tests can return canned
// HTTP responses deterministically. The Azure SDK's generated fake server races on the
// 202 async-delete path ("send on closed channel"), so we drive the pipeline directly
// instead.
type transportFunc func(req *http.Request) (*http.Response, error)

func (f transportFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func httpResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func defaultCapacityReservation() armcompute.CapacityReservation {
	return armcompute.CapacityReservation{
		Location: pointerutils.ToPtr("eastus"),
		SKU:      &armcompute.SKU{Name: pointerutils.ToPtr("Standard_D2s_v3"), Capacity: pointerutils.ToPtr[int64](1)},
	}
}
