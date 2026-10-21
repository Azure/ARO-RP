package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/portal/assets"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
)

// TestIndexPrometheusNonce ensures the server-rendered Prometheus UI opts its
// single legitimate inline <script> into the Content-Security-Policy via the
// per-request nonce, so that a strict script-src does not break the UI while
// still blocking cluster-authored inline script.
func TestIndexPrometheusNonce(t *testing.T) {
	tp := NewTestPortal(nil, nil, nil)

	assetPrometheus, err := assets.EmbeddedFiles.ReadFile("prometheus-ui/index.html")
	if err != nil {
		t.Fatal(err)
	}

	tp.p.templatePrometheus, err = template.New("index.html").Parse(string(assetPrometheus))
	if err != nil {
		t.Fatal(err)
	}

	const nonce = "test-nonce-value"
	r := httptest.NewRequest(http.MethodGet, "/subscriptions/x/resourcegroups/y/providers/microsoft.redhatopenshift/openshiftclusters/z/prometheus/", nil)
	r = r.WithContext(context.WithValue(r.Context(), middleware.ContextKeyCSPNonce, nonce))

	w := httptest.NewRecorder()
	tp.p.indexPrometheus(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(body), `nonce="`+nonce+`"`) {
		t.Errorf("rendered Prometheus index does not carry the CSP nonce on its inline script:\n%s", string(body))
	}
}
