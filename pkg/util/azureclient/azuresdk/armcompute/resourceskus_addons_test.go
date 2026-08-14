package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
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

	// The semaphore must be released even though we broke out of the
	// iteration early - a subsequent call must not deadlock.
	var gotAgain []string
	for sku, err := range c.List(context.Background(), "location eq eastus", false) {
		require.NoError(t, err)
		gotAgain = append(gotAgain, *sku.Name)
	}
	require.Equal(t, []string{"sku1", "sku2", "sku3"}, gotAgain)
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

	// The semaphore must be released even though the call errored - a
	// subsequent call (using a fresh client) must not deadlock.
	sku1 := &armcompute.ResourceSKU{Name: pointerutils.ToPtr("sku1"), ResourceType: pointerutils.ToPtr("virtualMachines")}
	srv2 := &fakearmcompute.ResourceSKUsServer{
		NewListPager: func(options *armcompute.ResourceSKUsClientListOptions) (resp fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]) {
			pagerResponse := fakeazcore.PagerResponder[armcompute.ResourceSKUsClientListResponse]{}
			pagerResponse.AddPage(http.StatusOK, armcompute.ResourceSKUsClientListResponse{
				ResourceSKUsResult: armcompute.ResourceSKUsResult{Value: []*armcompute.ResourceSKU{sku1}},
			}, nil)
			return pagerResponse
		},
	}
	c2 := newTestClient(t, srv2)

	var gotAgain []string
	for sku, err := range c2.List(context.Background(), "location eq eastus", false) {
		require.NoError(t, err)
		gotAgain = append(gotAgain, *sku.Name)
	}
	require.Equal(t, []string{"sku1"}, gotAgain)
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

	type result struct {
		name string
		err  error
	}

	doneA := make(chan result, 1)
	go func() {
		var r result
		for sku, err := range cA.List(context.Background(), "location eq eastus", false) {
			if err != nil {
				r.err = err
				break
			}
			r.name = *sku.Name
		}
		doneA <- r
	}()

	// Wait until A's request has actually reached the transport, i.e. it is
	// holding listSem.
	select {
	case <-btA.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for A to enter the transport")
	}

	doneB := make(chan result, 1)
	go func() {
		var r result
		for sku, err := range cB.List(context.Background(), "location eq eastus", false) {
			if err != nil {
				r.err = err
				break
			}
			r.name = *sku.Name
		}
		doneB <- r
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
	case r := <-doneA:
		require.NoError(t, r.err)
		require.Equal(t, "skuA", r.name)
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
	case r := <-doneB:
		require.NoError(t, r.err)
		require.Equal(t, "skuB", r.name)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for B to finish")
	}
}

// TestList_ContextCanceledWhileWaiting proves a caller queued behind
// listSem gives up promptly when its ctx is canceled, instead of being
// stuck until it is finally granted the semaphore.
func TestList_ContextCanceledWhileWaiting(t *testing.T) {
	// Hold listSem by starting a real List() call that blocks inside the
	// transport. Once bt.entered is closed we know the holder has acquired
	// listSem and is blocked, so the waiter below is guaranteed to queue.
	holder, bt := newBlockingTestClient(t, oneSKUServer("sku1"))
	holderDone := make(chan struct{})
	go func() {
		defer close(holderDone)
		for range holder.List(context.Background(), "location eq eastus", false) {
			break
		}
	}()
	select {
	case <-bt.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for holder to acquire listSem")
	}

	// Now start the waiter. Because the holder is blocking inside the
	// transport (and therefore holds listSem), the waiter is guaranteed to
	// be queued on listSem.Acquire at this point.
	waiter := newTestClient(t, oneSKUServer("sku2"))
	ctx, cancel := context.WithCancel(context.Background())

	type result struct {
		err  error
		took time.Duration
	}
	waiterDone := make(chan result, 1)
	start := time.Now()
	go func() {
		var gotErr error
		for _, err := range waiter.List(ctx, "location eq eastus", false) {
			gotErr = err
			break
		}
		waiterDone <- result{err: gotErr, took: time.Since(start)}
	}()

	// Cancel the waiter's context. It should return promptly without waiting
	// for the holder to release listSem.
	cancel()

	select {
	case r := <-waiterDone:
		require.Error(t, r.err)
		require.ErrorIs(t, r.err, context.Canceled)
		// The waiter should return promptly on cancellation, not be stuck
		// until the holder releases listSem.
		require.Less(t, r.took, 2*time.Second)
	case <-time.After(5 * time.Second):
		t.Fatal("List did not return after ctx was canceled while queued on listSem")
	}

	// Unblock the holder so it can exit cleanly.
	close(bt.release)
	<-holderDone
}
