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
)

const (
	// EnvFaultInjectFirst activates deterministic first-fail injection when set to a
	// comma-separated list of ARM error scenario names (see faultScenarios below).
	// Each unique write request URL is injected exactly once; subsequent requests to the
	// same URL (i.e. SDK retries) pass through to ARM. Scenarios rotate across distinct URLs.
	// Data-plane endpoints (blob, keyvault, etc.) are never injected.
	// Scenarios with a verbs list only inject on matching HTTP methods (e.g. 409 write-conflict
	// scenarios are restricted to PUT/POST/PATCH/DELETE since ARM never returns them on reads).
	// Scenario names rotate in order. Example: "ConflictingConcurrentWriteNotAllowed,TooManyRequests"
	// Applies to ARM management-plane clients (ArmClientOptions + DecorateSenderWithLogging) only.
	EnvFaultInjectFirst = "ARO_ARM_FAULT_FIRST"
)

// faultScenario describes a synthetic ARM error response matching a real ARM transient error.
type faultScenario struct {
	name   string // must match the map key in faultScenarios
	status int
	code   string
	msg    string
	// retryAfter sets the Retry-After header, exercising the header-based detection path.
	retryAfter bool
	// verbs restricts injection to the listed HTTP methods. Empty means any method.
	verbs []string
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

// firstFailLogOnce is shared between NewFirstFailPolicy and NewFirstFailSendDecorator.
// Both log the same startup message, so one log per process is sufficient.
var firstFailLogOnce sync.Once

type firstFailPolicy struct {
	scenarios []faultScenario
	host      string // if empty, defaults to "management.azure.com"
	mu        sync.Mutex
	injected  map[string]struct{} // set of "METHOD:URL" keys already injected
	sceneIdx  int                 // index of the next scenario to use; always accessed under mu
}

// NewFirstFailPolicy reads ARO_ARM_FAULT_FIRST and returns a policy that injects a synthetic ARM
// error on the first write to each distinct URL, letting retries to the same URL pass through.
// Returns nil when the env var is absent or contains no valid scenario names.
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

// Do injects a synthetic error on the first write to each distinct URL. Subsequent requests
// to the same URL (i.e. SDK retries) always pass through. Read verbs are never injected.
// Scenarios rotate across distinct injected URLs.
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
	// Check per-scenario verb filter before consuming the injection slot, so
	// a scenario with a narrower verb set doesn't silently burn a slot.
	if len(sc.verbs) > 0 && !slices.Contains(sc.verbs, req.Raw().Method) {
		p.mu.Unlock()
		return req.Next()
	}
	p.injected[key] = struct{}{}
	p.sceneIdx++
	p.mu.Unlock()
	logrus.Warnf("fault injected: %s (%d) on %s %s", sc.name, sc.status, req.Raw().Method, req.Raw().URL)
	resp := scenarioResponse(sc)
	resp.Request = req.Raw() // preserve request context for downstream handlers
	return resp, nil
}

// NewFirstFailSendDecorator applies the same first-fail logic as NewFirstFailPolicy for autorest
// senders. Returns nil when the env var is absent.
// IMPORTANT: apply only to management-plane senders; unlike firstFailPolicy there is no host
// filter; correctness relies on DecorateSenderWithLogging being wired to management-plane clients only.
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
			// Check per-scenario verb filter before consuming the injection slot,
			// so a scenario with a narrower verb set doesn't silently burn a slot.
			if len(sc.verbs) > 0 && !slices.Contains(sc.verbs, r.Method) {
				mu.Unlock()
				return s.Do(r)
			}
			injected[key] = struct{}{}
			sceneIdx++
			mu.Unlock()
			logrus.Warnf("fault injected: %s (%d) on %s %s", sc.name, sc.status, r.Method, r.URL)
			resp := scenarioResponse(sc)
			resp.Request = r // createPollingTracker dispatches on resp.Request.Method; nil → panic
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
