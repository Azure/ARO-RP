package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	fakeazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	fakearmcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7/fake"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const testSubscriptionID = "00000000-0000-0000-0000-000000000000"

func newTestClient(t *testing.T, srv *fakearmcompute.ResourceSKUsServer) ResourceSKUsClient {
	t.Helper()

	options := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: fakearmcompute.NewResourceSKUsServerTransport(srv),
		},
	}

	c, err := NewResourceSKUsClient(testSubscriptionID, &fakeazcore.TokenCredential{}, options)
	require.NoError(t, err)

	return c
}

// rawQueryCapturingTransport wraps another transport, additionally
// capturing each request's raw (still-encoded) query string. The fake ARM
// server transport decodes query parameters before we can observe them, so
// this lets tests assert on the literal bytes sent over the wire (e.g.
// whether spaces are encoded as "%20" or "+").
type rawQueryCapturingTransport struct {
	policy.Transporter
	rawQueries []string
}

func (t *rawQueryCapturingTransport) Do(req *http.Request) (*http.Response, error) {
	t.rawQueries = append(t.rawQueries, req.URL.RawQuery)
	return t.Transporter.Do(req)
}

func newTestClientWithRawQueryCapture(t *testing.T, srv *fakearmcompute.ResourceSKUsServer) (ResourceSKUsClient, *rawQueryCapturingTransport) {
	t.Helper()

	capture := &rawQueryCapturingTransport{Transporter: fakearmcompute.NewResourceSKUsServerTransport(srv)}
	options := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: capture,
		},
	}

	c, err := NewResourceSKUsClient(testSubscriptionID, &fakeazcore.TokenCredential{}, options)
	require.NoError(t, err)

	return c, capture
}

func TestList_HappyPath(t *testing.T) {
	sku1 := &armcompute.ResourceSKU{Name: pointerutils.ToPtr("sku1"), ResourceType: pointerutils.ToPtr("virtualMachines")}
	sku2 := &armcompute.ResourceSKU{Name: pointerutils.ToPtr("sku2"), ResourceType: pointerutils.ToPtr("virtualMachines")}

	srv := &fakearmcompute.ResourceSKUsServer{
		NewListPager: func(options *armcompute.ResourceSKUsClientListOptions) (resp fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]) {
			pagerResponse := fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]{}
			pagerResponse.AddPage(http.StatusOK, armcompute.ResourceSKUsClientListResponse{
				ResourceSKUsResult: armcompute.ResourceSKUsResult{
					Value: []*armcompute.ResourceSKU{sku1, sku2},
				},
			}, nil)
			return pagerResponse
		},
	}

	c := newTestClient(t, srv)

	var got []string
	for sku, err := range c.List(context.Background(), "location eq eastus", false) {
		require.NoError(t, err)
		got = append(got, *sku.Name)
	}

	require.Equal(t, []string{"sku1", "sku2"}, got)
}

func TestList_Pagination(t *testing.T) {
	sku1 := &armcompute.ResourceSKU{Name: pointerutils.ToPtr("sku1"), ResourceType: pointerutils.ToPtr("virtualMachines")}
	sku2 := &armcompute.ResourceSKU{Name: pointerutils.ToPtr("sku2"), ResourceType: pointerutils.ToPtr("virtualMachines")}

	srv := &fakearmcompute.ResourceSKUsServer{
		NewListPager: func(options *armcompute.ResourceSKUsClientListOptions) (resp fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]) {
			pagerResponse := fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]{}
			pagerResponse.AddPage(http.StatusOK, armcompute.ResourceSKUsClientListResponse{
				ResourceSKUsResult: armcompute.ResourceSKUsResult{Value: []*armcompute.ResourceSKU{sku1}},
			}, nil)
			pagerResponse.AddPage(http.StatusOK, armcompute.ResourceSKUsClientListResponse{
				ResourceSKUsResult: armcompute.ResourceSKUsResult{Value: []*armcompute.ResourceSKU{sku2}},
			}, nil)
			return pagerResponse
		},
	}

	c := newTestClient(t, srv)

	var got []string
	for sku, err := range c.List(context.Background(), "location eq eastus", false) {
		require.NoError(t, err)
		got = append(got, *sku.Name)
	}

	require.Equal(t, []string{"sku1", "sku2"}, got)
}

func TestList_EarlyStop(t *testing.T) {
	sku1 := &armcompute.ResourceSKU{Name: pointerutils.ToPtr("sku1"), ResourceType: pointerutils.ToPtr("virtualMachines")}
	sku2 := &armcompute.ResourceSKU{Name: pointerutils.ToPtr("sku2"), ResourceType: pointerutils.ToPtr("virtualMachines")}
	sku3 := &armcompute.ResourceSKU{Name: pointerutils.ToPtr("sku3"), ResourceType: pointerutils.ToPtr("virtualMachines")}

	srv := &fakearmcompute.ResourceSKUsServer{
		NewListPager: func(options *armcompute.ResourceSKUsClientListOptions) (resp fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]) {
			pagerResponse := fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]{}
			pagerResponse.AddPage(http.StatusOK, armcompute.ResourceSKUsClientListResponse{
				ResourceSKUsResult: armcompute.ResourceSKUsResult{Value: []*armcompute.ResourceSKU{sku1, sku2, sku3}},
			}, nil)
			return pagerResponse
		},
	}

	c := newTestClient(t, srv)

	// Only take the first SKU, then stop - the iterator must not decode (or
	// require the caller to have seen) sku2/sku3.
	var got []string
	for sku, err := range c.List(context.Background(), "location eq eastus", false) {
		require.NoError(t, err)
		got = append(got, *sku.Name)
		break
	}

	require.Equal(t, []string{"sku1"}, got)
}

func TestList_ErrorResponse(t *testing.T) {
	srv := &fakearmcompute.ResourceSKUsServer{
		NewListPager: func(options *armcompute.ResourceSKUsClientListOptions) (resp fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]) {
			pagerResponse := fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]{}
			pagerResponse.AddResponseError(http.StatusForbidden, "AuthorizationFailed")
			return pagerResponse
		},
	}

	c := newTestClient(t, srv)

	var gotErr error
	for _, err := range c.List(context.Background(), "location eq eastus", false) {
		if err != nil {
			gotErr = err
			break
		}
	}

	require.Error(t, gotErr)
	require.True(t, strings.Contains(gotErr.Error(), "403") || strings.Contains(gotErr.Error(), "AuthorizationFailed"))
}

func TestList_Filter(t *testing.T) {
	var gotFilter, gotIncludeExtendedLocations string

	srv := &fakearmcompute.ResourceSKUsServer{
		NewListPager: func(options *armcompute.ResourceSKUsClientListOptions) (resp fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]) {
			if options != nil {
				if options.Filter != nil {
					gotFilter = *options.Filter
				}
				if options.IncludeExtendedLocations != nil {
					gotIncludeExtendedLocations = *options.IncludeExtendedLocations
				}
			}
			pagerResponse := fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]{}
			pagerResponse.AddPage(http.StatusOK, armcompute.ResourceSKUsClientListResponse{}, nil)
			return pagerResponse
		},
	}

	c, capture := newTestClientWithRawQueryCapture(t, srv)

	for range c.List(context.Background(), "location eq westus2", true) {
	}

	require.Equal(t, "location eq westus2", gotFilter)
	require.Equal(t, "true", gotIncludeExtendedLocations)

	// The $filter value contains spaces; confirm they're sent as "%20"
	// (matching the generated ARM clients), not url.Values.Encode()'s
	// default '+'.
	require.Len(t, capture.rawQueries, 1)
	require.Contains(t, capture.rawQueries[0], "location%20eq%20westus2")
	require.NotContains(t, capture.rawQueries[0], "+")
}

// blockingTransport is a policy.Transporter that signals entered when Do is
// called, then blocks until release is closed before delegating to next.
// This lets tests observe/control exactly when a request is "in flight"
// without relying on wall-clock sleeps.
type blockingTransport struct {
	next     policy.Transporter
	entered  chan struct{}
	release  chan struct{}
	enterErr sync.Once
}

func (t *blockingTransport) Do(req *http.Request) (*http.Response, error) {
	t.enterErr.Do(func() { close(t.entered) })
	<-t.release
	return t.next.Do(req)
}

func newBlockingTestClient(t *testing.T, srv *fakearmcompute.ResourceSKUsServer) (ResourceSKUsClient, *blockingTransport) {
	t.Helper()

	bt := &blockingTransport{
		next:    fakearmcompute.NewResourceSKUsServerTransport(srv),
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	options := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: bt,
		},
	}

	c, err := NewResourceSKUsClient(testSubscriptionID, &fakeazcore.TokenCredential{}, options)
	require.NoError(t, err)

	return c, bt
}

func oneSKUServer(name string) *fakearmcompute.ResourceSKUsServer {
	sku := &armcompute.ResourceSKU{Name: pointerutils.ToPtr(name), ResourceType: pointerutils.ToPtr("virtualMachines")}
	return &fakearmcompute.ResourceSKUsServer{
		NewListPager: func(options *armcompute.ResourceSKUsClientListOptions) (resp fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]) {
			pagerResponse := fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]{}
			pagerResponse.AddPage(http.StatusOK, armcompute.ResourceSKUsClientListResponse{
				ResourceSKUsResult: armcompute.ResourceSKUsResult{Value: []*armcompute.ResourceSKU{sku}},
			}, nil)
			return pagerResponse
		},
	}
}

// TestList_SerializesConcurrentCalls proves listSem actually limits List()
// to one in-flight call at a time: a second call must not reach the
// transport until the first has fully finished iterating.
func TestList_SerializesConcurrentCalls(t *testing.T) {
	cA, btA := newBlockingTestClient(t, oneSKUServer("skuA"))
	cB, btB := newBlockingTestClient(t, oneSKUServer("skuB"))

	doneA := make(chan struct{})
	go func() {
		defer close(doneA)
		for sku, err := range cA.List(context.Background(), "location eq eastus", false) {
			require.NoError(t, err)
			require.Equal(t, "skuA", *sku.Name)
		}
	}()

	// Wait until A's request has actually reached the transport, i.e. it is
	// holding listSem.
	select {
	case <-btA.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for A to enter the transport")
	}

	doneB := make(chan struct{})
	go func() {
		defer close(doneB)
		for sku, err := range cB.List(context.Background(), "location eq eastus", false) {
			require.NoError(t, err)
			require.Equal(t, "skuB", *sku.Name)
		}
	}()

	// B should not be able to reach its transport while A holds listSem,
	// even though B's own request is otherwise ready to send.
	select {
	case <-btB.entered:
		t.Fatal("B reached the transport while A was still in flight")
	case <-time.After(200 * time.Millisecond):
	}

	// Release A and let it finish; only then should B be allowed through.
	close(btA.release)
	select {
	case <-doneA:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for A to finish")
	}

	select {
	case <-btB.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for B to enter the transport after A finished")
	}
	close(btB.release)

	select {
	case <-doneB:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for B to finish")
	}
}

// TestList_ContextCanceledWhileWaiting proves a caller queued behind
// listSem gives up promptly when its ctx is canceled, instead of being
// stuck until it is finally granted the semaphore.
func TestList_ContextCanceledWhileWaiting(t *testing.T) {
	// Simulate another in-flight List() call by holding listSem ourselves.
	require.NoError(t, listSem.Acquire(context.Background(), 1))
	defer listSem.Release(1)

	c := newTestClient(t, oneSKUServer("sku1"))

	ctx, cancel := context.WithCancel(context.Background())

	type result struct {
		err  error
		took time.Duration
	}
	resultCh := make(chan result, 1)
	start := time.Now()
	go func() {
		var gotErr error
		for _, err := range c.List(ctx, "location eq eastus", false) {
			gotErr = err
			break
		}
		resultCh <- result{err: gotErr, took: time.Since(start)}
	}()

	// Give the goroutine a moment to actually start waiting on
	// listSem.Acquire before canceling.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case r := <-resultCh:
		require.Error(t, r.err)
		require.True(t, errors.Is(r.err, context.Canceled))
		// The waiter should return promptly on cancellation, not be stuck
		// until listSem is released (which we deliberately never do here).
		require.Less(t, r.took, 2*time.Second)
	case <-time.After(5 * time.Second):
		t.Fatal("List did not return after ctx was canceled while queued on listSem")
	}
}
