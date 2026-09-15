package cosmosdb

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/ugorji/go/codec"

	pkg "github.com/Azure/ARO-RP/pkg/api"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const (
	// startPosition is the continuation the iterators under test begin at. It
	// is not the empty string, so that "did not advance" and "was reset to the
	// beginning of the feed" can be told apart.
	startPosition = "page-1"

	// nextPosition is the Etag a response carries, naming the page after the
	// one being read.
	nextPosition = "page-2"

	// undecodableBody is a response body which fails to decode, as a page
	// containing a document sealed with a key the process does not hold does.
	undecodableBody = "{"
)

// newTestChangeFeedIterator returns an iterator reading from hostname, and the
// hook capturing what it logs.
func newTestChangeFeedIterator(t *testing.T, maxFailures int, hostname string, hc *http.Client) (*resilientChangeFeedIterator[pkg.OpenShiftClusterDocuments], *test.Hook) {
	t.Helper()

	h, log := testlog.New()

	i := newResilientChangeFeedIterator[pkg.OpenShiftClusterDocuments](
		&databaseClient{
			log:              log,
			jsonHandle:       &codec.JsonHandle{},
			databaseHostname: hostname,
			hc:               hc,
			maxRetries:       1,
		},
		"dbs/testdb/colls/testcoll",
		&Options{Continuation: startPosition},
		// The generated setOptions does nothing for a change feed read, which
		// passes no document and no triggers.
		func(*Options, http.Header) error { return nil },
	)
	i.maxFailures = maxFailures

	return i, h
}

// newTestServer starts a TLS server and returns its hostname and a client
// configured to trust it.
func newTestServer(t *testing.T, handler http.HandlerFunc) (string, *http.Client) {
	t.Helper()

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	return strings.TrimPrefix(server.URL, "https://"), server.Client()
}

// nextExpectingFailure calls Next and fails the test if it succeeds.
func nextExpectingFailure(t *testing.T, i *resilientChangeFeedIterator[pkg.OpenShiftClusterDocuments], attempt int) {
	t.Helper()

	if _, err := i.Next(context.Background(), 10); err == nil {
		t.Fatalf("iterator.Next() error = nil on attempt %d, want an error", attempt)
	}
}

// TestChangeFeedAdvancesPastAnUnreadablePage covers the wedge this iterator
// exists to prevent: a page which fails to decode on every attempt.
func TestChangeFeedAdvancesPastAnUnreadablePage(t *testing.T) {
	const maxFailures = 3

	hostname, hc := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Etag", nextPosition)
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, undecodableBody)
	})

	i, h := newTestChangeFeedIterator(t, maxFailures, hostname, hc)

	for attempt := 1; attempt < maxFailures; attempt++ {
		nextExpectingFailure(t, i, attempt)

		if got := i.Continuation(); got != startPosition {
			t.Errorf("iterator.Continuation() = %q after %d of %d tolerated failures, want %q", got, attempt, maxFailures, startPosition)
		}
		if got := i.PagesSkipped(); got != 0 {
			t.Errorf("iterator.PagesSkipped() = %d after %d of %d tolerated failures, want 0", got, attempt, maxFailures)
		}
	}

	nextExpectingFailure(t, i, maxFailures)

	if got := i.Continuation(); got != nextPosition {
		t.Errorf("iterator.Continuation() = %q after %d failures, want %q", got, maxFailures, nextPosition)
	}
	if len(h.Entries) != 1 {
		t.Errorf("logged %d entries, want 1 reporting the skipped page", len(h.Entries))
	}
	// The skip must remain countable after the fact. consecutiveFailures is
	// reset by the advance, so without this the only surviving evidence that a
	// page was never delivered is the log line above.
	if got := i.PagesSkipped(); got != 1 {
		t.Errorf("iterator.PagesSkipped() = %d after a page was skipped, want 1", got)
	}
}

// TestChangeFeedDoesNotAdvanceWithoutAPosition is the most important test here.
// An empty If-None-Match asks Cosmos DB to read from the beginning of the feed,
// so advancing to an absent Etag would re-read the entire collection and arrive
// back where it started. The iterator must stay put instead.
func TestChangeFeedDoesNotAdvanceWithoutAPosition(t *testing.T) {
	const maxFailures = 3

	for _, tt := range []struct {
		name    string
		handler http.HandlerFunc
		// transport reaches a port nothing is listening on, so that hc.Do
		// fails and no response reaches do at all.
		transport bool
	}{
		{
			name:      "no response at all",
			transport: true,
		},
		{
			name: "response carries no Etag",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				io.WriteString(w, `{"code": "InternalServerError", "message": "fake error"}`)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var (
				hostname string
				hc       *http.Client
			)
			if tt.transport {
				hostname, hc = "127.0.0.1:1", &http.Client{}
			} else {
				hostname, hc = newTestServer(t, tt.handler)
			}

			i, _ := newTestChangeFeedIterator(t, maxFailures, hostname, hc)

			// Well past the threshold, to catch an advance that is merely late.
			for attempt := 1; attempt <= maxFailures*3; attempt++ {
				nextExpectingFailure(t, i, attempt)

				if got := i.Continuation(); got != startPosition {
					t.Fatalf("iterator.Continuation() = %q after %d failures, want %q", got, attempt, startPosition)
				}
			}
		})
	}
}

// TestChangeFeedResetsItsFailureCountOnSuccess checks that the threshold
// measures persistent failure rather than accumulated failure, so that a page
// is never skipped because of faults spread across an otherwise healthy feed.
func TestChangeFeedResetsItsFailureCountOnSuccess(t *testing.T) {
	const (
		maxFailures     = 3
		positionAfterOK = "page-a"
		positionAfterKO = "page-b"
	)

	attempts := 0

	hostname, hc := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++

		w.Header().Set("Content-Type", "application/json")

		switch {
		case attempts < maxFailures:
			// Short of the threshold, then interrupted by a success.
			w.Header().Set("Etag", nextPosition)
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, undecodableBody)
		case attempts == maxFailures:
			w.Header().Set("Etag", positionAfterOK)
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{}`)
		default:
			w.Header().Set("Etag", positionAfterKO)
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, undecodableBody)
		}
	})

	i, _ := newTestChangeFeedIterator(t, maxFailures, hostname, hc)

	for attempt := 1; attempt < maxFailures; attempt++ {
		nextExpectingFailure(t, i, attempt)
	}

	if _, err := i.Next(context.Background(), 10); err != nil {
		t.Fatalf("iterator.Next() error = %v on the successful attempt, want nil", err)
	}
	if got := i.Continuation(); got != positionAfterOK {
		t.Fatalf("iterator.Continuation() = %q after a successful read, want %q", got, positionAfterOK)
	}

	// Had the count not been reset, the first of these would take the iterator
	// past the threshold and advance it.
	for attempt := 1; attempt < maxFailures; attempt++ {
		nextExpectingFailure(t, i, attempt)

		if got := i.Continuation(); got != positionAfterOK {
			t.Errorf("iterator.Continuation() = %q after %d failures following a success, want %q", got, attempt, positionAfterOK)
		}
	}

	nextExpectingFailure(t, i, maxFailures)

	if got := i.Continuation(); got != positionAfterKO {
		t.Errorf("iterator.Continuation() = %q after %d consecutive failures, want %q", got, maxFailures, positionAfterKO)
	}
}

// TestChangeFeedReportsAStallPeriodically checks that a feed which can neither
// read nor advance says so more than once, but not on every poll. The incident
// which prompted this produced 5.49 million identical log lines.
func TestChangeFeedReportsAStallPeriodically(t *testing.T) {
	const maxFailures = 3

	hostname, hc := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"code": "InternalServerError", "message": "fake error"}`)
	})

	i, h := newTestChangeFeedIterator(t, maxFailures, hostname, hc)

	const polls = maxFailures * 4
	for attempt := 1; attempt <= polls; attempt++ {
		nextExpectingFailure(t, i, attempt)
	}

	if want := polls / maxFailures; len(h.Entries) != want {
		t.Errorf("logged %d entries over %d failed polls, want %d", len(h.Entries), polls, want)
	}
}
