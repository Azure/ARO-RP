package cosmosdb

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strconv"

	pkg "github.com/Azure/ARO-RP/pkg/api"
)

// defaultChangeFeedMaxFailures is how many consecutive failures at the same
// position are tolerated before a resilient change feed gives up on the page
// and moves on.
//
// The consumers poll every ten seconds, so thirty failures is five minutes of
// uninterrupted failure. That is far longer than any throttling or transient
// service fault, and so short against the time a stalled feed can otherwise go
// unnoticed that the choice is not delicate.
const defaultChangeFeedMaxFailures = 30

// A resilientChangeFeedIterator is a change feed iterator which can make
// progress past a page it cannot read.
//
// The generated iterators cannot. Their continuation advances only after a
// successful read, so a page which fails deterministically — because a document
// in it cannot be decoded, say, because the key which sealed it is not held —
// is re-requested from the same position on every poll, for as long as the
// process runs. Nothing downstream of the feed is then updated again.
//
// This type re-implements Next with that error path corrected. It is
// hand-written because the generated files must not be edited; if the change is
// taken upstream into the generator, this file can be deleted.
type resilientChangeFeedIterator[T any] struct {
	client     *databaseClient
	path       string
	options    *Options
	setOptions func(*Options, http.Header) error

	continuation        string
	consecutiveFailures int
	maxFailures         int
}

func newResilientChangeFeedIterator[T any](client *databaseClient, path string, options *Options, setOptions func(*Options, http.Header) error) *resilientChangeFeedIterator[T] {
	continuation := ""
	if options != nil {
		continuation = options.Continuation
	}

	return &resilientChangeFeedIterator[T]{
		client:       client,
		path:         path,
		options:      options,
		setOptions:   setOptions,
		continuation: continuation,
		maxFailures:  defaultChangeFeedMaxFailures,
	}
}

func (i *resilientChangeFeedIterator[T]) Next(ctx context.Context, maxItemCount int) (*T, error) {
	headers := http.Header{}
	headers.Set("A-IM", "Incremental feed")
	headers.Set("X-Ms-Max-Item-Count", strconv.Itoa(maxItemCount))
	if i.continuation != "" {
		headers.Set("If-None-Match", i.continuation)
	}

	err := i.setOptions(i.options, headers)
	if err != nil {
		return nil, err
	}

	var docs *T
	err = i.client.do(ctx, http.MethodGet, i.path+"/docs", "docs", i.path, http.StatusOK, nil, &docs, headers)
	if IsErrorStatusCode(err, http.StatusNotModified) {
		err = nil
	}
	if err != nil {
		i.onFailure(headers, err)
		return nil, err
	}

	i.consecutiveFailures = 0
	i.continuation = headers.Get("Etag")

	return docs, nil
}

// onFailure records a failed read and, once the same position has failed
// maxFailures times in succession, advances past it.
//
// The position to advance to is already in hand. do copies the response headers
// into the caller's map whenever there was a response at all, and _do returns
// the response alongside a decode error, so the Etag naming the page after this
// one is present even though the read failed.
//
// It is absent when there was no response — a transport error, for instance —
// which is why the advance is conditional on it. An empty If-None-Match asks
// Cosmos DB to read the feed from the beginning, so advancing to "" would reset
// the feed, re-read the whole collection, and arrive back at the same page.
// The guard is load-bearing rather than defensive.
//
// Requiring maxFailures consecutive failures, and resetting the count on any
// success, means the threshold measures persistence: a page is only ever
// skipped when it cannot be read at all, never during a passing fault.
func (i *resilientChangeFeedIterator[T]) onFailure(headers http.Header, err error) {
	i.consecutiveFailures++

	// Acting on every maxFailures'th failure rather than on every failure past
	// the first threshold keeps a feed which cannot advance from logging on
	// every poll, while still having it report itself periodically.
	if i.consecutiveFailures%i.maxFailures != 0 {
		return
	}

	etag := headers.Get("Etag")
	if etag == "" || etag == i.continuation {
		i.client.log.Errorf("changefeed %s: %d consecutive failures with no position to advance to; the feed is stalled: %s", i.path, i.consecutiveFailures, err)
		return
	}

	i.client.log.Errorf("changefeed %s: advancing past a page which failed %d times in succession; its contents will not be delivered: %s", i.path, i.consecutiveFailures, err)

	i.continuation = etag
	i.consecutiveFailures = 0
}

func (i *resilientChangeFeedIterator[T]) Continuation() string {
	return i.continuation
}

// NewResilientOpenShiftClusterDocumentChangeFeed returns a change feed iterator
// which can make progress past a page it cannot read. See
// resilientChangeFeedIterator.
func NewResilientOpenShiftClusterDocumentChangeFeed(c OpenShiftClusterDocumentClient, options *Options) OpenShiftClusterDocumentIterator {
	cc, ok := c.(*openShiftClusterDocumentClient)
	if !ok {
		// Not a real Cosmos DB client — the fake, in all current cases. It has
		// no failing page to get past.
		return c.ChangeFeed(options)
	}

	return newResilientChangeFeedIterator[pkg.OpenShiftClusterDocuments](
		cc.databaseClient, cc.path, options,
		func(options *Options, headers http.Header) error {
			return cc.setOptions(options, nil, headers)
		},
	)
}

// NewResilientSubscriptionDocumentChangeFeed returns a change feed iterator
// which can make progress past a page it cannot read. See
// resilientChangeFeedIterator.
func NewResilientSubscriptionDocumentChangeFeed(c SubscriptionDocumentClient, options *Options) SubscriptionDocumentIterator {
	cc, ok := c.(*subscriptionDocumentClient)
	if !ok {
		return c.ChangeFeed(options)
	}

	return newResilientChangeFeedIterator[pkg.SubscriptionDocuments](
		cc.databaseClient, cc.path, options,
		func(options *Options, headers http.Header) error {
			return cc.setOptions(options, nil, headers)
		},
	)
}

// NewResilientGatewayDocumentChangeFeed returns a change feed iterator which
// can make progress past a page it cannot read. See
// resilientChangeFeedIterator.
func NewResilientGatewayDocumentChangeFeed(c GatewayDocumentClient, options *Options) GatewayDocumentIterator {
	cc, ok := c.(*gatewayDocumentClient)
	if !ok {
		return c.ChangeFeed(options)
	}

	return newResilientChangeFeedIterator[pkg.GatewayDocuments](
		cc.databaseClient, cc.path, options,
		func(options *Options, headers http.Header) error {
			return cc.setOptions(options, nil, headers)
		},
	)
}
