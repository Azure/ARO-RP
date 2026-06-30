package common

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"

	testhttp "github.com/Azure/ARO-RP/test/util/http/server"
)

// mustHost extracts the host (host:port) from a URL string for use in firstFailPolicy.host.
func mustHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u.Host
}

func TestParseScenarios(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNames []string
	}{
		{name: "empty returns nil", input: "", wantNames: nil},
		{
			name:      "single known scenario",
			input:     "TooManyRequests",
			wantNames: []string{"TooManyRequests"},
		},
		{
			name:      "multiple known scenarios",
			input:     "ConflictingConcurrentWriteNotAllowed,TooManyRequests",
			wantNames: []string{"ConflictingConcurrentWriteNotAllowed", "TooManyRequests"},
		},
		{
			name:      "whitespace trimmed",
			input:     " TooManyRequests , RetryableError ",
			wantNames: []string{"TooManyRequests", "RetryableError"},
		},
		{
			name:      "unknown names skipped",
			input:     "NotARealScenario,TooManyRequests",
			wantNames: []string{"TooManyRequests"},
		},
		{
			name:      "all unknown returns nil",
			input:     "FakeScenario,AnotherFake",
			wantNames: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseScenarios(tt.input)
			if len(got) != len(tt.wantNames) {
				t.Fatalf("parseScenarios(%q) returned %d scenarios, want %d", tt.input, len(got), len(tt.wantNames))
			}
			for i, sc := range got {
				if sc.name != tt.wantNames[i] {
					t.Errorf("scenario %d: got name %q, want %q", i, sc.name, tt.wantNames[i])
				}
			}
		})
	}
}

func TestNewFirstFailPolicy(t *testing.T) {
	t.Run("returns nil when env var unset", func(t *testing.T) {
		t.Setenv(EnvFaultInjectFirst, "")
		if got := NewFirstFailPolicy(); got != nil {
			t.Errorf("expected nil, got %T", got)
		}
	})

	t.Run("returns nil for unknown scenarios only", func(t *testing.T) {
		t.Setenv(EnvFaultInjectFirst, "NotAScenario")
		if got := NewFirstFailPolicy(); got != nil {
			t.Errorf("expected nil, got %T", got)
		}
	})

	t.Run("returns policy for known scenario", func(t *testing.T) {
		t.Setenv(EnvFaultInjectFirst, "TooManyRequests")
		if got := NewFirstFailPolicy(); got == nil {
			t.Error("expected non-nil policy")
		}
	})
}

func TestScenarioResponse(t *testing.T) {
	tests := []struct {
		scenario        string
		wantStatus      int
		wantRetryAfter  bool
		wantCode        string
		wantMsgContains string
	}{
		{
			scenario:        "ConflictingConcurrentWriteNotAllowed",
			wantStatus:      http.StatusConflict,
			wantRetryAfter:  true,
			wantCode:        "ConflictingConcurrentWriteNotAllowed",
			wantMsgContains: "Please retry later",
		},
		{
			scenario:        "CanceledAndSupersededDueToAnotherOperation",
			wantStatus:      http.StatusConflict,
			wantRetryAfter:  true,
			wantCode:        "CanceledAndSupersededDueToAnotherOperation",
			wantMsgContains: "Please retry later",
		},
		{
			scenario:        "TooManyRequests",
			wantStatus:      http.StatusTooManyRequests,
			wantRetryAfter:  true,
			wantCode:        "TooManyRequests",
			wantMsgContains: "Please retry later",
		},
		{
			scenario:        "RetryableError",
			wantStatus:      http.StatusTooManyRequests,
			wantRetryAfter:  true,
			wantCode:        "RetryableError",
			wantMsgContains: "RetryableError",
		},
	}
	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			sc, ok := faultScenarios[tt.scenario]
			if !ok {
				t.Fatalf("scenario %q not found in faultScenarios", tt.scenario)
			}
			resp := scenarioResponse(sc)

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("StatusCode = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			retryAfterVal := resp.Header.Get("Retry-After")
			if (retryAfterVal != "") != tt.wantRetryAfter {
				t.Errorf("Retry-After present = %v, want %v", retryAfterVal != "", tt.wantRetryAfter)
			}
			if tt.wantRetryAfter && retryAfterVal != "1" {
				t.Errorf("Retry-After = %q, want \"1\"", retryAfterVal)
			}

			// Verify the body is valid ARM error JSON with the expected code.
			var armErr struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&armErr); err != nil {
				t.Fatalf("body is not valid ARM JSON: %v", err)
			}
			if armErr.Error.Code != tt.wantCode {
				t.Errorf("error.code = %q, want %q", armErr.Error.Code, tt.wantCode)
			}
			if tt.wantMsgContains != "" && !strings.Contains(armErr.Error.Message, tt.wantMsgContains) {
				t.Errorf("error.message = %q, want it to contain %q", armErr.Error.Message, tt.wantMsgContains)
			}
		})
	}
}

// TestFirstFailPolicyDo verifies per-URL inject-once behavior.
func TestFirstFailPolicyDo(t *testing.T) {
	newPolicy := func(scenarios []faultScenario, srvHost string) *firstFailPolicy {
		return &firstFailPolicy{scenarios: scenarios, host: srvHost, injected: make(map[string]struct{})}
	}

	t.Run("first write to a URL injects; retry to same URL passes; scenarios rotate across URLs", func(t *testing.T) {
		scenarios := []faultScenario{
			faultScenarios["ConflictingConcurrentWriteNotAllowed"],
			faultScenarios["TooManyRequests"],
		}

		srv, close := testhttp.NewServer()
		defer close()
		// 2 pass-through retries reach the server
		srv.AppendResponse(testhttp.WithStatusCode(http.StatusOK))
		srv.AppendResponse(testhttp.WithStatusCode(http.StatusOK))

		p := newPolicy(scenarios, mustHost(srv.URL()))
		pl := runtime.NewPipeline("test", "v1.0.0", runtime.PipelineOptions{
			PerCall: []policy.Policy{p},
		}, &policy.ClientOptions{Transport: srv})

		do := func(method, rawURL string, wantStatus int) {
			t.Helper()
			req, err := runtime.NewRequest(context.Background(), method, rawURL)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := pl.Do(req)
			if err != nil {
				t.Fatalf("%s %s: unexpected error: %v", method, rawURL, err)
			}
			if resp.StatusCode != wantStatus {
				t.Errorf("%s %s: got %d, want %d", method, rawURL, resp.StatusCode, wantStatus)
			}
			resp.Body.Close()
		}

		urlA := srv.URL() + "?resource=a"
		urlB := srv.URL() + "?resource=b"

		do(http.MethodPut, urlA, http.StatusConflict)        // URL_A: inject (scenario 0)
		do(http.MethodPut, urlA, http.StatusOK)              // URL_A: retry passes through
		do(http.MethodPut, urlB, http.StatusTooManyRequests) // URL_B: inject (scenario 1)
		do(http.MethodPut, urlB, http.StatusOK)              // URL_B: retry passes through

		if got := srv.Requests(); got != 2 {
			t.Errorf("server received %d requests, want 2 (only retries pass through)", got)
		}
	})

	t.Run("GET: passes through without touching injection state", func(t *testing.T) {
		srv, close := testhttp.NewServer()
		defer close()
		for i := 0; i < 4; i++ {
			srv.AppendResponse(testhttp.WithStatusCode(http.StatusOK))
		}

		pol := newPolicy([]faultScenario{faultScenarios["ConflictingConcurrentWriteNotAllowed"]}, mustHost(srv.URL()))
		pl := runtime.NewPipeline("test", "v1.0.0", runtime.PipelineOptions{
			PerCall: []policy.Policy{pol},
		}, &policy.ClientOptions{Transport: srv})

		// GETs are never injected and do not affect the per-URL injection map.
		for i := 0; i < 4; i++ {
			req, err := runtime.NewRequest(context.Background(), http.MethodGet, srv.URL())
			if err != nil {
				t.Fatal(err)
			}
			resp, err := pl.Do(req)
			if err != nil {
				t.Fatalf("call %d: unexpected error: %v", i+1, err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("call %d: got %d, want 200", i+1, resp.StatusCode)
			}
			resp.Body.Close()
		}
		if got := srv.Requests(); got != 4 {
			t.Errorf("server received %d requests, want 4 (all GETs pass through)", got)
		}
	})

	t.Run("concurrent writes: each distinct URL injected exactly once regardless of interleaving", func(t *testing.T) {
		// Simulates multiple goroutines writing to different resources simultaneously
		// (e.g. creating several IPs in parallel). Each URL gets exactly one injection;
		// retries to the same URL always pass through, even when other URLs are also
		// being written concurrently.
		srv, close := testhttp.NewServer()
		defer close()
		// 4 distinct URLs × 1 retry each = 4 server responses
		for i := 0; i < 4; i++ {
			srv.AppendResponse(testhttp.WithStatusCode(http.StatusOK))
		}

		p := newPolicy([]faultScenario{faultScenarios["TooManyRequests"]}, mustHost(srv.URL()))
		pl := runtime.NewPipeline("test", "v1.0.0", runtime.PipelineOptions{
			PerCall: []policy.Policy{p},
		}, &policy.ClientOptions{Transport: srv})

		put := func(rawURL string, wantStatus int) {
			t.Helper()
			req, err := runtime.NewRequest(context.Background(), http.MethodPut, rawURL)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := pl.Do(req)
			if err != nil {
				t.Fatalf("PUT %s: unexpected error: %v", rawURL, err)
			}
			if resp.StatusCode != wantStatus {
				t.Errorf("PUT %s: got %d, want %d", rawURL, resp.StatusCode, wantStatus)
			}
			resp.Body.Close()
		}

		urls := []string{
			srv.URL() + "?ip=a",
			srv.URL() + "?ip=b",
			srv.URL() + "?ip=c",
			srv.URL() + "?ip=d",
		}

		// First call to each URL: inject
		for _, u := range urls {
			put(u, http.StatusTooManyRequests)
		}
		// Retry to each URL: pass through (URL already marked)
		for _, u := range urls {
			put(u, http.StatusOK)
		}

		if got := srv.Requests(); got != 4 {
			t.Errorf("server received %d requests, want 4 (only retries pass through)", got)
		}
	})
}

func TestNewFirstFailSendDecorator(t *testing.T) {
	t.Run("returns nil when env var unset", func(t *testing.T) {
		t.Setenv(EnvFaultInjectFirst, "")
		if got := NewFirstFailSendDecorator(); got != nil {
			t.Errorf("expected nil, got non-nil")
		}
	})

	t.Run("returns nil for unknown scenarios only", func(t *testing.T) {
		t.Setenv(EnvFaultInjectFirst, "NotAScenario")
		if got := NewFirstFailSendDecorator(); got != nil {
			t.Errorf("expected nil, got non-nil")
		}
	})

	t.Run("returns non-nil for known scenario", func(t *testing.T) {
		t.Setenv(EnvFaultInjectFirst, "TooManyRequests")
		if got := NewFirstFailSendDecorator(); got == nil {
			t.Error("expected non-nil decorator")
		}
	})
}

// TestFirstFailSendDecoratorDo verifies the autorest SendDecorator injects HTTP responses (not
// errors) on the first write to each URL and passes through retries to the same URL. Returning
// a response (not an error) lets autorest's DoRetryForStatusCodes retry 429s automatically.
func TestFirstFailSendDecoratorDo(t *testing.T) {
	t.Setenv(EnvFaultInjectFirst, "ConflictingConcurrentWriteNotAllowed,TooManyRequests")

	srv, close := testhttp.NewServer()
	defer close()
	// 2 distinct URLs × 1 retry each = 2 server responses
	srv.AppendResponse(testhttp.WithStatusCode(http.StatusOK))
	srv.AppendResponse(testhttp.WithStatusCode(http.StatusOK))

	d := NewFirstFailSendDecorator()
	if d == nil {
		t.Fatal("expected non-nil decorator")
	}
	decorated := d(srv)

	url1 := srv.URL() + "?r=1"
	url2 := srv.URL() + "?r=2"

	type wantCall struct {
		url        string
		wantStatus int // expected HTTP status code (no errors returned)
	}
	calls := []wantCall{
		{url1, http.StatusConflict},        // first write to url1: inject 409 (ConflictingConcurrentWriteNotAllowed)
		{url1, http.StatusOK},              // retry to url1: pass-through → 200
		{url2, http.StatusTooManyRequests}, // first write to url2: inject 429 (TooManyRequests)
		{url2, http.StatusOK},              // retry to url2: pass-through → 200
	}
	for i, c := range calls {
		req, err := http.NewRequest(http.MethodPut, c.url, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := decorated.Do(req)
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i+1, err)
		}
		if resp.StatusCode != c.wantStatus {
			t.Errorf("call %d: got status %d, want %d", i+1, resp.StatusCode, c.wantStatus)
		}
		resp.Body.Close()
	}
	if got := srv.Requests(); got != 2 {
		t.Errorf("server received %d requests, want 2 (only retries pass through)", got)
	}
}
