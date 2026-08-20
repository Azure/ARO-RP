package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/sync/semaphore"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

// resourceSKUsAPIVersion mirrors the generated client's api-version.
const resourceSKUsAPIVersion = "2021-07-01"

// listSem caps List() to one in-flight call per process. Package-level
// since callers construct a new client per request.
var listSem = semaphore.NewWeighted(1)

type ResourceSKUsClientAddons interface {
	List(ctx context.Context, filter string, includeExtendedLocations bool) iter.Seq2[armcompute.ResourceSKU, error]
}

// resourceSKUsPage is the pager state for List; SKUs are streamed to the
// caller as decoded (see streamResourceSKUsPage), not stored on the page.
type resourceSKUsPage struct {
	nextLink string
}

// List returns an iterator over the Resource SKUs matching filter.
// Unlike the generated NewListPager, pages are streamed to the caller as
// decoded rather than fully buffered. Calls are serialised via listSem.
func (c *resourceSKUsClient) List(ctx context.Context, filter string, includeExtendedLocations bool) iter.Seq2[armcompute.ResourceSKU, error] {
	ex := "false"
	if includeExtendedLocations {
		ex = "true"
	}

	return func(yield func(armcompute.ResourceSKU, error) bool) {
		if err := listSem.Acquire(ctx, 1); err != nil {
			yield(armcompute.ResourceSKU{}, err)
			return
		}
		defer listSem.Release(1)

		// stopped ends pagination even if the last page had a nextLink.
		stopped := false

		pager := runtime.NewPager(runtime.PagingHandler[resourceSKUsPage]{
			More: func(page resourceSKUsPage) bool {
				return !stopped && page.nextLink != ""
			},
			Fetcher: func(ctx context.Context, page *resourceSKUsPage) (resourceSKUsPage, error) {
				nextLink := ""
				if page != nil {
					nextLink = page.nextLink
				}

				resp, err := runtime.FetcherForNextLink(ctx, c.armClient.Pipeline(), nextLink, func(ctx context.Context) (*policy.Request, error) {
					return c.listCreateRequest(ctx, filter, ex)
				}, &runtime.FetcherForNextLinkOptions{
					NextReq: func(ctx context.Context, nextLink string) (*policy.Request, error) {
						req, err := runtime.NewRequest(ctx, http.MethodGet, nextLink)
						if err != nil {
							return nil, err
						}
						skipBodyDownload(req)
						return req, nil
					},
				})
				if err != nil {
					return resourceSKUsPage{}, err
				}

				next, cont, err := streamResourceSKUsPage(resp.Body, yield)
				if err != nil {
					return resourceSKUsPage{}, fmt.Errorf("decoding resource SKUs page: %w", err)
				}
				if !cont {
					stopped = true
				}
				return resourceSKUsPage{nextLink: next}, nil
			},
			// Match the generated pager's tracing tags.
			Tracer: c.armClient.Tracer(),
		})

		for pager.More() {
			if _, err := pager.NextPage(ctx); err != nil {
				yield(armcompute.ResourceSKU{}, err)
				return
			}
		}
	}
}

// skipBodyDownload stops the pipeline from fully buffering the response
// body, so streamResourceSKUsPage can decode it as bytes arrive.
func skipBodyDownload(req *policy.Request) {
	runtime.SkipBodyDownload(req)
}

// listCreateRequest builds the request for the first page, mirroring the
// generated ResourceSKUsClient.listCreateRequest. Subsequent pages use the
// NextReq closure passed to runtime.FetcherForNextLink above.
func (c *resourceSKUsClient) listCreateRequest(ctx context.Context, filter, includeExtendedLocations string) (*policy.Request, error) {
	if c.subscriptionID == "" {
		return nil, errors.New("parameter subscriptionID cannot be empty")
	}

	urlPath := strings.ReplaceAll(
		"/subscriptions/{subscriptionId}/providers/Microsoft.Compute/skus",
		"{subscriptionId}", url.PathEscape(c.subscriptionID),
	)

	req, err := runtime.NewRequest(ctx, http.MethodGet, runtime.JoinPaths(c.armClient.Endpoint(), urlPath))
	if err != nil {
		return nil, err
	}

	reqQP := req.Raw().URL.Query()
	if filter != "" {
		reqQP.Set("$filter", filter)
	}
	reqQP.Set("api-version", resourceSKUsAPIVersion)
	reqQP.Set("includeExtendedLocations", includeExtendedLocations)
	// Match generated ARM clients' %20 space encoding, not '+' (see
	// openshiftversions_client.go).
	req.Raw().URL.RawQuery = strings.ReplaceAll(reqQP.Encode(), "+", "%20")
	req.Raw().Header["Accept"] = []string{"application/json"}

	skipBodyDownload(req)

	return req, nil
}

// streamResourceSKUsPage decodes one page, yielding SKUs one at a time
// instead of buffering the whole page. Returns nextLink and whether to keep
// iterating; false means yield asked to stop and the rest is left unread.
func streamResourceSKUsPage(body io.ReadCloser, yield func(armcompute.ResourceSKU, error) bool) (nextLink string, cont bool, err error) {
	defer body.Close()

	dec := json.NewDecoder(body)

	if _, err := dec.Token(); err != nil { // consume opening `{`
		return "", false, err
	}

	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return "", false, err
		}
		key, _ := keyTok.(string)

		switch key {
		case "value":
			if _, err := dec.Token(); err != nil { // consume opening `[`
				return "", false, err
			}
			for dec.More() {
				var sku armcompute.ResourceSKU
				if err := dec.Decode(&sku); err != nil {
					return "", false, err
				}
				if !yield(sku, nil) {
					return "", false, nil
				}
			}
			if _, err := dec.Token(); err != nil { // consume closing `]`
				return "", false, err
			}
		case "nextLink":
			if err := dec.Decode(&nextLink); err != nil {
				return "", false, err
			}
		default:
			// Skip any field we don't care about, without decoding it into
			// a typed value.
			var discard json.RawMessage
			if err := dec.Decode(&discard); err != nil {
				return "", false, err
			}
		}
	}

	if _, err := dec.Token(); err != nil { // consume closing `}`
		return "", false, err
	}

	return nextLink, true, nil
}
