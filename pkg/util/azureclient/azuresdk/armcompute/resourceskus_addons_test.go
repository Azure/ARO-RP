package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	fakeazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

const testSubscriptionID = "00000000-0000-0000-0000-000000000000"

// skuPage is the wire shape of one ARM ResourceSKUs list page.
type skuPage struct {
	Value    []armcompute.ResourceSKU `json:"value"`
	NextLink string                   `json:"nextLink,omitempty"`
}

// newTestServer starts an httptest.TLSServer whose handler is provided by the
// caller. It registers a cleanup to close the server when the test ends.
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// newTestClient creates a ResourceSKUsClient pointed at srv, trusting its
// self-signed TLS certificate.
func newTestClient(t *testing.T, srv *httptest.Server) ResourceSKUsClient {
	t.Helper()

	certPool := x509.NewCertPool()
	for _, c := range srv.TLS.Certificates {
		leaf, err := x509.ParseCertificate(c.Certificate[0])
		require.NoError(t, err)
		certPool.AddCert(leaf)
	}

	transport := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: certPool},
	}}

	cloudCfg := cloud.Configuration{
		ActiveDirectoryAuthorityHost: srv.URL,
		Services: map[cloud.ServiceName]cloud.ServiceConfiguration{
			cloud.ResourceManager: {
				Audience: srv.URL,
				Endpoint: srv.URL,
			},
		},
	}

	options := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud:     cloudCfg,
			Transport: transport,
			Retry:     policy.RetryOptions{MaxRetries: 0},
		},
	}

	c, err := NewResourceSKUsClient(testSubscriptionID, &fakeazcore.TokenCredential{}, options)
	require.NoError(t, err)
	return c
}

// servePages returns an http.HandlerFunc that serves the given pages in order.
// nextLink URLs are injected lazily using the incoming request's host, so the
// handler does not need to know the server address at construction time.
func servePages(pages []skuPage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idx := 0
		if p := r.URL.Query().Get("page"); p != "" {
			fmt.Sscanf(p, "%d", &idx) //nolint:errcheck
		}
		if idx < 0 || idx >= len(pages) {
			http.Error(w, "bad page index", http.StatusBadRequest)
			return
		}
		page := pages[idx]
		if idx < len(pages)-1 {
			page.NextLink = fmt.Sprintf(
				"https://%s/subscriptions/%s/providers/Microsoft.Compute/skus?api-version=%s&page=%d",
				r.Host, testSubscriptionID, resourceSKUsAPIVersion, idx+1,
			)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(page) //nolint:errcheck
	}
}

// oneSKUPage returns a single-page slice containing one SKU with the given name.
func oneSKUPage(name string) []skuPage {
	return []skuPage{{
		Value: []armcompute.ResourceSKU{
			{Name: strPtr(name), ResourceType: strPtr("virtualMachines")},
		},
	}}
}

func strPtr(s string) *string { return &s }

// TestList_HappyPath verifies that all SKUs on a single page are yielded.
func TestList_HappyPath(t *testing.T) {
	srv := newTestServer(t, servePages([]skuPage{{
		Value: []armcompute.ResourceSKU{
			{Name: strPtr("sku1"), ResourceType: strPtr("virtualMachines")},
			{Name: strPtr("sku2"), ResourceType: strPtr("virtualMachines")},
		},
	}}))

	var got []string
	for sku, err := range newTestClient(t, srv).List(context.Background(), "location eq eastus", false) {
		require.NoError(t, err)
		got = append(got, *sku.Name)
	}
	require.Equal(t, []string{"sku1", "sku2"}, got)
}

// TestList_Pagination verifies that SKUs across multiple pages are all yielded.
func TestList_Pagination(t *testing.T) {
	srv := newTestServer(t, servePages([]skuPage{
		{Value: []armcompute.ResourceSKU{{Name: strPtr("sku1"), ResourceType: strPtr("virtualMachines")}}},
		{Value: []armcompute.ResourceSKU{{Name: strPtr("sku2"), ResourceType: strPtr("virtualMachines")}}},
	}))

	var got []string
	for sku, err := range newTestClient(t, srv).List(context.Background(), "location eq eastus", false) {
		require.NoError(t, err)
		got = append(got, *sku.Name)
	}
	require.Equal(t, []string{"sku1", "sku2"}, got)
}

// TestList_EarlyStop verifies that breaking out of iteration releases the
// semaphore, allowing a subsequent call to proceed.
func TestList_EarlyStop(t *testing.T) {
	srv := newTestServer(t, servePages([]skuPage{{
		Value: []armcompute.ResourceSKU{
			{Name: strPtr("sku1"), ResourceType: strPtr("virtualMachines")},
			{Name: strPtr("sku2"), ResourceType: strPtr("virtualMachines")},
			{Name: strPtr("sku3"), ResourceType: strPtr("virtualMachines")},
		},
	}}))

	c := newTestClient(t, srv)

	var got []string
	for sku, err := range c.List(context.Background(), "location eq eastus", false) {
		require.NoError(t, err)
		got = append(got, *sku.Name)
		break
	}
	require.Equal(t, []string{"sku1"}, got)

	// The semaphore must be released even though we broke early — a
	// subsequent call must not deadlock.
	var gotAgain []string
	for sku, err := range c.List(context.Background(), "location eq eastus", false) {
		require.NoError(t, err)
		gotAgain = append(gotAgain, *sku.Name)
	}
	require.Equal(t, []string{"sku1", "sku2", "sku3"}, gotAgain)
}

// TestList_ErrorResponse verifies that an HTTP error is surfaced as an
// iterator error and that the semaphore is released afterwards.
func TestList_ErrorResponse(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":"AuthorizationFailed"}}`, http.StatusForbidden)
	})

	var gotErr error
	for _, err := range newTestClient(t, srv).List(context.Background(), "location eq eastus", false) {
		if err != nil {
			gotErr = err
			break
		}
	}
	require.Error(t, gotErr)
	require.True(t, strings.Contains(gotErr.Error(), "403") || strings.Contains(gotErr.Error(), "AuthorizationFailed"))

	// The semaphore must be released after an error — a subsequent call
	// (on a fresh client pointed at a healthy server) must not deadlock.
	srv2 := newTestServer(t, servePages(oneSKUPage("sku1")))

	var gotAgain []string
	for sku, err := range newTestClient(t, srv2).List(context.Background(), "location eq eastus", false) {
		require.NoError(t, err)
		gotAgain = append(gotAgain, *sku.Name)
	}
	require.Equal(t, []string{"sku1"}, gotAgain)
}

// TestList_Filter verifies that the filter and includeExtendedLocations query
// parameters are sent correctly, and that spaces are encoded as %20 not +.
func TestList_Filter(t *testing.T) {
	var gotRawQuery string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(skuPage{}) //nolint:errcheck
	})

	for range newTestClient(t, srv).List(context.Background(), "location eq westus2", true) {
	}

	require.Contains(t, gotRawQuery, "location%20eq%20westus2")
	require.Contains(t, gotRawQuery, "includeExtendedLocations=true")
	require.NotContains(t, gotRawQuery, "+")
}

// TestList_NextLinkBeforeValue verifies that streamResourceSKUsPage handles
// ARM responses where nextLink appears before the value array in the JSON
// object — a field ordering the SDK fake transport never produces.
func TestList_NextLinkBeforeValue(t *testing.T) {
	page2 := skuPage{
		Value: []armcompute.ResourceSKU{
			{Name: strPtr("sku2"), ResourceType: strPtr("virtualMachines")},
		},
	}

	calls := 0
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		calls++
		if calls == 1 {
			// Write page 1 with nextLink before value — a field ordering
			// that json.Encoder never produces but ARM sometimes does.
			nextLink := fmt.Sprintf(
				"https://%s/subscriptions/%s/providers/Microsoft.Compute/skus?api-version=%s&page=1",
				r.Host, testSubscriptionID, resourceSKUsAPIVersion,
			)
			fmt.Fprintf(w, //nolint:errcheck
				`{"nextLink":%q,"value":[{"name":"sku1","resourceType":"virtualMachines"}]}`,
				nextLink,
			)
		} else {
			json.NewEncoder(w).Encode(page2) //nolint:errcheck
		}
	})

	var got []string
	for sku, err := range newTestClient(t, srv).List(context.Background(), "location eq eastus", false) {
		require.NoError(t, err)
		got = append(got, *sku.Name)
	}
	require.Equal(t, []string{"sku1", "sku2"}, got)
}

// blockingTransport signals entered when the first request arrives, then
// blocks until release is closed. This lets tests control exactly when a
// request is "in flight" without relying on wall-clock sleeps.
type blockingTransport struct {
	next     policy.Transporter
	entered  chan struct{}
	release  chan struct{}
	enterErr sync.Once
}

func (bt *blockingTransport) Do(req *http.Request) (*http.Response, error) {
	bt.enterErr.Do(func() { close(bt.entered) })
	<-bt.release
	return bt.next.Do(req)
}

// newBlockingTestClient creates a client that will block inside the transport
// until the returned blockingTransport.release channel is closed.
func newBlockingTestClient(t *testing.T, srv *httptest.Server) (ResourceSKUsClient, *blockingTransport) {
	t.Helper()

	certPool := x509.NewCertPool()
	for _, c := range srv.TLS.Certificates {
		leaf, err := x509.ParseCertificate(c.Certificate[0])
		require.NoError(t, err)
		certPool.AddCert(leaf)
	}

	inner := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: certPool},
	}}

	bt := &blockingTransport{
		next:    inner,
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}

	cloudCfg := cloud.Configuration{
		ActiveDirectoryAuthorityHost: srv.URL,
		Services: map[cloud.ServiceName]cloud.ServiceConfiguration{
			cloud.ResourceManager: {
				Audience: srv.URL,
				Endpoint: srv.URL,
			},
		},
	}

	options := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud:     cloudCfg,
			Transport: bt,
			Retry:     policy.RetryOptions{MaxRetries: 0},
		},
	}

	c, err := NewResourceSKUsClient(testSubscriptionID, &fakeazcore.TokenCredential{}, options)
	require.NoError(t, err)
	return c, bt
}

// TestList_SerializesConcurrentCalls proves listSem limits List() to one
// in-flight call at a time: a second call must not reach the transport until
// the first has fully finished iterating.
func TestList_SerializesConcurrentCalls(t *testing.T) {
	srvA := newTestServer(t, servePages(oneSKUPage("skuA")))
	srvB := newTestServer(t, servePages(oneSKUPage("skuB")))

	cA, btA := newBlockingTestClient(t, srvA)
	cB, btB := newBlockingTestClient(t, srvB)

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

	// Wait until A's request has reached the transport, meaning it holds listSem.
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

	// B must not reach its transport while A holds listSem.
	select {
	case <-btB.entered:
		t.Fatal("B reached the transport while A was still in flight")
	case <-time.After(200 * time.Millisecond):
	}

	// Release A; only then should B be allowed through.
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

// TestList_ContextCanceledWhileWaiting proves that a caller queued behind
// listSem returns promptly when its context is canceled.
func TestList_ContextCanceledWhileWaiting(t *testing.T) {
	holderSrv := newTestServer(t, servePages(oneSKUPage("sku1")))
	holder, bt := newBlockingTestClient(t, holderSrv)

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

	waiterSrv := newTestServer(t, servePages(oneSKUPage("sku2")))
	waiter := newTestClient(t, waiterSrv)
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

	cancel()

	select {
	case r := <-waiterDone:
		require.Error(t, r.err)
		require.ErrorIs(t, r.err, context.Canceled)
		require.Less(t, r.took, 2*time.Second)
	case <-time.After(5 * time.Second):
		t.Fatal("List did not return after ctx was canceled while queued on listSem")
	}

	close(bt.release)
	<-holderDone
}
