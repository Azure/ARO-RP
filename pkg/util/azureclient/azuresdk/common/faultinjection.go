package common

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/go-autorest/autorest"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

const (
	// EnvFaultInjectFirst activates fault injection when set to a comma-separated list of
	// scenario names (see faultScenarios). Each distinct write URL is injected exactly once;
	// retries to the same URL pass through. Applies to management-plane clients only
	// (ArmClientOptions + DecorateSenderWithLogging); data-plane endpoints are never injected.
	EnvFaultInjectFirst = "ARO_ARM_FAULT_FIRST"
)

type faultScenario struct {
	name       string // must match the map key in faultScenarios
	status     int
	code       string
	msg        string
	retryAfter bool
	verbs      []string // HTTP methods to inject on; empty means any method
}

var faultScenarios = map[string]faultScenario{
	// verbs: write-conflict 409s never occur on GET/HEAD.
	"ConflictingConcurrentWriteNotAllowed": {
		name:       "ConflictingConcurrentWriteNotAllowed",
		status:     http.StatusConflict,
		code:       "ConflictingConcurrentWriteNotAllowed",
		msg:        "The operation was interrupted by a conflicting concurrent write on the same entity. Please retry later.",
		retryAfter: true,
		verbs:      []string{http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete},
	},
	// verbs: write-conflict 409s never occur on GET/HEAD.
	"CanceledAndSupersededDueToAnotherOperation": {
		name:       "CanceledAndSupersededDueToAnotherOperation",
		status:     http.StatusConflict,
		code:       "CanceledAndSupersededDueToAnotherOperation",
		msg:        "Operation was canceled due to a conflicting operation. Please retry later.",
		retryAfter: true,
		verbs:      []string{http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete},
	},
	// verbs: read 429s are not covered by this package's retry wrappers; only writes are retried.
	"TooManyRequests": {
		name:       "TooManyRequests",
		status:     http.StatusTooManyRequests,
		code:       "TooManyRequests",
		msg:        "Please retry later.",
		retryAfter: true,
		verbs:      []string{http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete},
	},
	"RetryableError": {
		name:       "RetryableError",
		status:     http.StatusTooManyRequests,
		code:       "RetryableError",
		msg:        "RetryableError: A retryable error occurred. Please retry later.",
		retryAfter: true,
		verbs:      []string{http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete},
	},
}

// firstFailLogOnce is shared between NewFirstFailPolicy and NewFirstFailSendDecorator
// so both log the same startup message only once per process.
var firstFailLogOnce sync.Once

type firstFailPolicy struct {
	scenarios []faultScenario
	host      string // if empty, defaults to "management.azure.com"
	mu        sync.Mutex
	injected  map[string]struct{} // "METHOD:URL" keys already injected
	sceneIdx  int                 // always accessed under mu
}

func NewFirstFailPolicy() policy.Policy {
	envVal := os.Getenv(EnvFaultInjectFirst)
	scenarios := parseScenarios(envVal)
	if len(scenarios) == 0 {
		return nil
	}
	logStartup(envVal, scenarios)
	return &firstFailPolicy{
		scenarios: scenarios,
		injected:  make(map[string]struct{}),
	}
}

func (p *firstFailPolicy) Do(req *policy.Request) (*http.Response, error) {
	targetHost := p.host
	if targetHost == "" {
		targetHost = "management.azure.com"
	}
	if req.Raw().URL.Host != targetHost {
		return req.Next()
	}

	if !slices.Contains([]string{http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete}, req.Raw().Method) {
		return req.Next()
	}

	key := req.Raw().Method + ":" + req.Raw().URL.String()

	p.mu.Lock()
	if _, done := p.injected[key]; done {
		p.mu.Unlock()
		return req.Next()
	}
	sc := p.scenarios[p.sceneIdx%len(p.scenarios)]
	// Check the verb filter before consuming the slot: a narrow-verb scenario must not
	// silently burn an injection slot on a method it doesn't cover.
	if len(sc.verbs) > 0 && !slices.Contains(sc.verbs, req.Raw().Method) {
		p.mu.Unlock()
		return req.Next()
	}
	p.injected[key] = struct{}{}
	p.sceneIdx++
	p.mu.Unlock()

	logrus.Warnf("fault injected: %s (%d) on %s %s", sc.name, sc.status, utillog.Sanitize(req.Raw().Method), utillog.Sanitize(req.Raw().URL.String()))
	resp := scenarioResponse(sc)
	resp.Request = req.Raw()
	return resp, nil
}

// lroPollFaultPolicy injects LRO failures on ARM call paths that return 202. It records
// the Azure-AsyncOperation poll URL from each 202 and returns a 200 Failed on the first
// GET to that URL. Each write URL is injected once; retries to the same URL pass through.
//
// Kept separate from firstFailPolicy: LRO injection must be tied to real 202 responses,
// not to arbitrary writes.
type lroPollFaultPolicy struct {
	scenarios      []faultScenario
	host           string // if empty, defaults to "management.azure.com"
	mu             sync.Mutex
	injectedWrites map[string]struct{}      // "METHOD:URL" write keys already injected; always accessed under mu
	pendingPolls   map[string]faultScenario // real poll URL to inject, keyed by URL string; always accessed under mu
	sceneIdx       int                      // always accessed under mu
}

func NewLROPollFaultPolicy() policy.Policy {
	envVal := os.Getenv(EnvFaultInjectFirst)
	scenarios := parseScenarios(envVal)
	if len(scenarios) == 0 {
		return nil
	}
	names := make([]string, len(scenarios))
	for i, sc := range scenarios {
		names[i] = sc.name
	}
	logrus.Warnf("ARM LRO poll injection enabled (%s=%s): rotating through %s",
		EnvFaultInjectFirst, envVal, strings.Join(names, ", "))
	return &lroPollFaultPolicy{
		scenarios:      scenarios,
		injectedWrites: make(map[string]struct{}),
		pendingPolls:   make(map[string]faultScenario),
	}
}

func (p *lroPollFaultPolicy) Do(req *policy.Request) (*http.Response, error) {
	targetHost := p.host
	if targetHost == "" {
		targetHost = "management.azure.com"
	}

	if req.Raw().URL.Host != targetHost {
		return req.Next()
	}

	if req.Raw().Method == http.MethodGet {
		pollURL := req.Raw().URL.String()
		p.mu.Lock()
		sc, pending := p.pendingPolls[pollURL]
		if pending {
			delete(p.pendingPolls, pollURL)
		}
		p.mu.Unlock()
		if pending {
			logrus.Warnf("fault injected: %s LRO poll failure on %s", sc.name, utillog.Sanitize(pollURL))
			resp := lroFailureResponse(sc)
			resp.Request = req.Raw()
			return resp, nil
		}
		return req.Next()
	}

	resp, err := req.Next()
	if err != nil || resp == nil {
		return resp, err
	}

	if resp.StatusCode == http.StatusAccepted && resp.Header.Get("Azure-AsyncOperation") != "" {
		writeKey := req.Raw().Method + ":" + req.Raw().URL.String()
		p.mu.Lock()
		if _, done := p.injectedWrites[writeKey]; !done {
			sc := p.scenarios[p.sceneIdx%len(p.scenarios)]
			p.sceneIdx++
			p.injectedWrites[writeKey] = struct{}{}
			pollURL := resp.Header.Get("Azure-AsyncOperation")
			p.pendingPolls[pollURL] = sc
			p.mu.Unlock()
			logrus.Warnf("fault pending: %s LRO poll injection set for %s", sc.name, utillog.Sanitize(pollURL))
		} else {
			p.mu.Unlock()
		}
	}

	return resp, nil
}

// NewFirstFailSendDecorator applies the same first-fail logic as NewFirstFailPolicy for autorest
// senders. Returns nil when the env var is absent.
// IMPORTANT: unlike firstFailPolicy there is no host filter; apply only to management-plane
// senders. Correctness relies on DecorateSenderWithLogging being wired to those clients only.
func NewFirstFailSendDecorator() autorest.SendDecorator {
	envVal := os.Getenv(EnvFaultInjectFirst)
	scenarios := parseScenarios(envVal)
	if len(scenarios) == 0 {
		return nil
	}
	logStartup(envVal, scenarios)
	var (
		mu       sync.Mutex
		injected = make(map[string]struct{})
		sceneIdx int
	)
	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
			if !slices.Contains([]string{http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete}, r.Method) {
				return s.Do(r)
			}
			key := r.Method + ":" + r.URL.String()
			mu.Lock()
			if _, done := injected[key]; done {
				mu.Unlock()
				return s.Do(r)
			}
			sc := scenarios[sceneIdx%len(scenarios)]
			// Check the verb filter before consuming the slot: a narrow-verb scenario must not
			// silently burn an injection slot on a method it doesn't cover.
			if len(sc.verbs) > 0 && !slices.Contains(sc.verbs, r.Method) {
				mu.Unlock()
				return s.Do(r)
			}
			injected[key] = struct{}{}
			sceneIdx++
			mu.Unlock()
			logrus.Warnf("fault injected: %s (%d) on %s %s", sc.name, sc.status, utillog.Sanitize(r.Method), utillog.Sanitize(r.URL.String()))
			resp := scenarioResponse(sc)
			resp.Request = r // autorest dispatches on resp.Request.Method; nil causes a panic
			return resp, nil
		})
	}
}

func logStartup(envVal string, scenarios []faultScenario) {
	firstFailLogOnce.Do(func() {
		names := make([]string, len(scenarios))
		for i, sc := range scenarios {
			names[i] = sc.name
		}
		logrus.Warnf("ARM first-fail injection enabled (%s=%s): rotating through %s",
			EnvFaultInjectFirst, envVal, strings.Join(names, ", "))
	})
}

func scenarioResponse(sc faultScenario) *http.Response {
	body := fmt.Sprintf(`{"error":{"code":%q,"message":%q}}`, sc.code, sc.msg)
	h := http.Header{}
	if sc.retryAfter {
		h.Set("Retry-After", "1")
	}
	return &http.Response{
		StatusCode: sc.status,
		Status:     fmt.Sprintf("%d %s", sc.status, http.StatusText(sc.status)),
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// lroFailureResponse returns a 200 OK with status=Failed, matching the ARM async operation
// terminal failure shape. A 200 (not 4xx) is correct: the poll itself succeeded; the operation failed.
func lroFailureResponse(sc faultScenario) *http.Response {
	body := fmt.Sprintf(`{"status":"Failed","error":{"code":%q,"message":%q}}`, sc.code, sc.msg)
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func parseScenarios(s string) []faultScenario {
	if s == "" {
		return nil
	}
	var out []faultScenario
	for _, name := range strings.Split(s, ",") {
		name = strings.TrimSpace(name)
		if sc, ok := faultScenarios[name]; ok {
			out = append(out, sc)
		} else {
			logrus.Warnf("%s: unknown scenario %q, skipping", EnvFaultInjectFirst, name)
		}
	}
	return out
}
