package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

// firstPortalUUID is the first UUID emitted by NewFakePortal()'s
// deterministic generator (namespace=PORTAL=3, counter=1).
const firstPortalUUID = "03030303-0303-0303-0303-030303030001"

func TestAdminOpenShiftClusterKubeconfigNew(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	resourcePath := strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName"))

	type test struct {
		name              string
		urlSuffix         string // "/kubeconfig/new" or "/kubeconfig/newelevated"
		systemDataHeader  string // raw JSON; empty => header omitted
		omitServingCert   bool   // simulate cert-load failure path
		injectPortalError error  // simulate a Cosmos write failure
		wantStatusCode    int
		wantError         string
		wantElevated      bool
		wantUsername      string
		wantFilenameSlug  string // expected filename body (no extension)
	}

	for _, tt := range []*test{
		{
			name:             "non-elevated kubeconfig issued by SRE",
			urlSuffix:        "/kubeconfig/new",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com","createdBy":"someone-else@redhat.com"}`,
			wantStatusCode:   http.StatusOK,
			wantElevated:     false,
			wantUsername:     "sre@redhat.com",
			wantFilenameSlug: "resourcename",
		},
		{
			name:             "elevated kubeconfig issued by SRE",
			urlSuffix:        "/kubeconfig/newelevated",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			wantStatusCode:   http.StatusOK,
			wantElevated:     true,
			wantUsername:     "sre@redhat.com",
			wantFilenameSlug: "resourcename-elevated",
		},
		{
			name:             "falls back to createdBy when lastModifiedBy is empty",
			urlSuffix:        "/kubeconfig/new",
			systemDataHeader: `{"createdBy":"creator@redhat.com"}`,
			wantStatusCode:   http.StatusOK,
			wantElevated:     false,
			wantUsername:     "creator@redhat.com",
			wantFilenameSlug: "resourcename",
		},
		{
			name:             "missing SystemData header produces an empty username (auditing gap surfaced by Geneva, not blocked here)",
			urlSuffix:        "/kubeconfig/new",
			systemDataHeader: "",
			wantStatusCode:   http.StatusOK,
			wantElevated:     false,
			wantUsername:     "",
			wantFilenameSlug: "resourcename",
		},
		{
			name:             "portal serving cert unavailable returns 503",
			urlSuffix:        "/kubeconfig/new",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			omitServingCert:  true,
			wantStatusCode:   http.StatusServiceUnavailable,
			wantError:        "503: InternalServerError: : Portal serving certificate is not available; the kubeconfig endpoint is disabled.",
		},
		{
			name:              "cosmos create failure surfaces 500 with no partial body",
			urlSuffix:         "/kubeconfig/new",
			systemDataHeader:  `{"lastModifiedBy":"sre@redhat.com"}`,
			injectPortalError: errors.New("simulated cosmos write failure"),
			wantStatusCode:    http.StatusInternalServerError,
			wantError:         "500: InternalServerError: : simulated cosmos write failure",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithPortal()
			defer ti.done()

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			// The shared test infra does not exercise the portal-keyvault
			// fetch path, so portalServingCert is nil after NewFrontend by
			// default. Inject the same self-signed cert that the shared
			// test infra uses for the RP listener so the happy paths can
			// run; leave it nil to exercise the 503 fallback.
			if !tt.omitServingCert {
				f.portalServingCert = servercerts[0]
			}

			if tt.injectPortalError != nil {
				ti.portalClient.SetError(tt.injectPortalError)
			}

			go f.Run(ctx, nil, nil)

			header := http.Header{}
			if tt.systemDataHeader != "" {
				header.Set("X-Ms-Arm-Resource-System-Data", tt.systemDataHeader)
			}

			resp, b, err := ti.request(http.MethodPost,
				"https://server/admin"+resourcePath+tt.urlSuffix,
				header, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Clear the fake's sticky error so the post-response
			// ListAll assertion isn't blocked by the same injection.
			if tt.injectPortalError != nil {
				ti.portalClient.SetError(nil)
			}

			// validateResponse rejects non-empty bodies when wantResponse
			// is nil, so it's only useful on the error path here; the
			// happy path body is a kubeconfig blob asserted below.
			if tt.wantError != "" {
				if err := validateResponse(resp, b, tt.wantStatusCode, tt.wantError, nil); err != nil {
					t.Error(err)
				}
				docs, listErr := ti.portalClient.ListAll(ctx, nil)
				if listErr != nil {
					t.Fatal(listErr)
				}
				if len(docs.PortalDocuments) != 0 {
					t.Errorf("expected 0 portal documents persisted, got %d", len(docs.PortalDocuments))
				}
				return
			}

			if resp.StatusCode != tt.wantStatusCode {
				t.Fatalf("unexpected status code %d, wanted %d: %s", resp.StatusCode, tt.wantStatusCode, string(b))
			}

			// Content-Disposition is what triggers the browser to save
			// the response as a file with the cluster's name.
			wantCD := fmt.Sprintf(`attachment; filename="%s.kubeconfig"`, tt.wantFilenameSlug)
			if got := resp.Header.Get("Content-Disposition"); got != wantCD {
				t.Errorf("Content-Disposition: got %q, want %q", got, wantCD)
			}

			// Body must be a valid clientcmdv1.Config that points the
			// kubectl client at the per-region admin.aro.azure.com host
			// and trusts the portal serving cert as the CA.
			var kc clientcmdv1.Config
			if err := json.Unmarshal(b, &kc); err != nil {
				t.Fatalf("response body is not a valid clientcmdv1.Config: %v\nbody: %s", err, string(b))
			}
			if len(kc.Clusters) != 1 {
				t.Fatalf("expected 1 cluster in kubeconfig, got %d", len(kc.Clusters))
			}
			wantServer := "https://eastus.admin.aro.azure.com" + resourcePath + "/kubeconfig/proxy"
			if got := kc.Clusters[0].Cluster.Server; got != wantServer {
				t.Errorf("kubeconfig server: got %q, want %q", got, wantServer)
			}
			wantCA := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercerts[0].Raw})
			if got := kc.Clusters[0].Cluster.CertificateAuthorityData; string(got) != string(wantCA) {
				t.Errorf("kubeconfig CA data does not match the injected portal serving cert")
			}
			if len(kc.AuthInfos) != 1 || kc.AuthInfos[0].AuthInfo.Token != firstPortalUUID {
				t.Errorf("kubeconfig token: got %+v, want %q", kc.AuthInfos, firstPortalUUID)
			}

			// Persisted PortalDocument: this is what the portal binary's
			// /kubeconfig/proxy/* reverse proxy reads to translate the
			// token back into the cluster ARM ID + elevation flag.
			docs, err := ti.portalClient.ListAll(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(docs.PortalDocuments) != 1 {
				t.Fatalf("expected 1 portal document persisted, got %d", len(docs.PortalDocuments))
			}
			doc := docs.PortalDocuments[0]
			if doc.ID != firstPortalUUID {
				t.Errorf("portal doc ID: got %q, want %q", doc.ID, firstPortalUUID)
			}
			if doc.Portal == nil {
				t.Fatal("portal doc has nil Portal payload")
			}
			if doc.Portal.ID != resourcePath {
				t.Errorf("portal doc Portal.ID: got %q, want %q", doc.Portal.ID, resourcePath)
			}
			if doc.Portal.Username != tt.wantUsername {
				t.Errorf("portal doc Portal.Username: got %q, want %q", doc.Portal.Username, tt.wantUsername)
			}
			if doc.Portal.Kubeconfig == nil {
				t.Fatal("portal doc Portal.Kubeconfig is nil")
			}
			if doc.Portal.Kubeconfig.Elevated != tt.wantElevated {
				t.Errorf("portal doc Kubeconfig.Elevated: got %v, want %v", doc.Portal.Kubeconfig.Elevated, tt.wantElevated)
			}
			if doc.TTL <= 0 {
				t.Errorf("portal doc TTL: got %d, want > 0", doc.TTL)
			}
		})
	}
}
