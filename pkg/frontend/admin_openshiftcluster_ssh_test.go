package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestAdminOpenShiftClusterSSHNewElevated(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	resourcePath := strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName"))

	// One host key per test run. Real deployments load this from the portal
	// keyvault; the test just needs any RSA pubkey the KnownHosts helper
	// can render.
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	hostPub, err := cryptossh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	expectedKnownHostLine := knownhosts.Line([]string{"eastus.admin.aro.azure.com"}, hostPub)

	type test struct {
		name              string
		systemDataHeader  string
		body              interface{}
		contentType       string // "" => header omitted
		omitHostKey       bool
		injectPortalError error
		wantStatusCode    int
		wantError         string
		wantUsername      string
		wantMaster        int
	}

	for _, tt := range []*test{
		{
			name:             "master 0 issued by SRE",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: 0},
			contentType:      "application/json",
			wantStatusCode:   http.StatusOK,
			wantUsername:     "sre@redhat.com",
			wantMaster:       0,
		},
		{
			name:             "master 2 issued by SRE",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: 2},
			contentType:      "application/json",
			wantStatusCode:   http.StatusOK,
			wantUsername:     "sre@redhat.com",
			wantMaster:       2,
		},
		{
			name:             "falls back to createdBy when lastModifiedBy is empty",
			systemDataHeader: `{"createdBy":"creator@redhat.com"}`,
			body:             &adminSSHRequest{Master: 1},
			contentType:      "application/json",
			wantStatusCode:   http.StatusOK,
			wantUsername:     "creator@redhat.com",
			wantMaster:       1,
		},
		{
			name:             "missing SystemData header produces empty username",
			systemDataHeader: "",
			body:             &adminSSHRequest{Master: 0},
			contentType:      "application/json",
			wantStatusCode:   http.StatusOK,
			wantUsername:     "",
			wantMaster:       0,
		},
		{
			name:             "portal host key unavailable returns 503",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: 0},
			contentType:      "application/json",
			omitHostKey:      true,
			wantStatusCode:   http.StatusServiceUnavailable,
			wantError:        "503: InternalServerError: : Portal SSH host key is not available; the SSH endpoint is disabled.",
		},
		{
			name:             "wrong Content-Type returns 415",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: 0},
			contentType:      "text/plain",
			wantStatusCode:   http.StatusUnsupportedMediaType,
			wantError:        "415: UnsupportedMediaType: : The content media type 'text/plain' is not supported. Only 'application/json' is supported.",
		},
		{
			name:             "master out of range returns 400",
			systemDataHeader: `{"lastModifiedBy":"sre@redhat.com"}`,
			body:             &adminSSHRequest{Master: 3},
			contentType:      "application/json",
			wantStatusCode:   http.StatusBadRequest,
			wantError:        "400: InvalidParameter: properties.master: master must be 0, 1, or 2.",
		},
		{
			name:              "cosmos create failure surfaces 500",
			systemDataHeader:  `{"lastModifiedBy":"sre@redhat.com"}`,
			body:              &adminSSHRequest{Master: 0},
			contentType:       "application/json",
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

			if !tt.omitHostKey {
				f.portalSSHHostPubKey = hostPub
			}

			if tt.injectPortalError != nil {
				ti.portalClient.SetError(tt.injectPortalError)
			}

			go f.Run(ctx, nil, nil)

			header := http.Header{}
			if tt.systemDataHeader != "" {
				header.Set("X-Ms-Arm-Resource-System-Data", tt.systemDataHeader)
			}
			if tt.contentType != "" {
				header.Set("Content-Type", tt.contentType)
			}

			resp, b, err := ti.request(http.MethodPost,
				"https://server/admin"+resourcePath+"/ssh/newelevated",
				header, tt.body)
			if err != nil {
				t.Fatal(err)
			}

			if tt.injectPortalError != nil {
				ti.portalClient.SetError(nil)
			}

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

			var got adminSSHResponse
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("response is not a valid adminSSHResponse: %v\nbody: %s", err, string(b))
			}
			if got.Password != firstPortalUUID {
				t.Errorf("response Password: got %q, want %q", got.Password, firstPortalUUID)
			}
			// Command must pin the injected host key; the LHS-of-@ username
			// is what the portal SSH proxy'll match on PasswordCallback.
			wantUserPrefix := strings.SplitN(tt.wantUsername, "@", 2)[0]
			if !strings.Contains(got.Command, expectedKnownHostLine) {
				t.Errorf("response Command missing KnownHosts pin\ncommand: %s\nwant substring: %s", got.Command, expectedKnownHostLine)
			}
			if !strings.Contains(got.Command, wantUserPrefix+"@eastus.admin.aro.azure.com") {
				t.Errorf("response Command missing %q@host suffix\ncommand: %s", wantUserPrefix, got.Command)
			}

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
			if doc.Portal.SSH == nil {
				t.Fatal("portal doc Portal.SSH is nil")
			}
			if doc.Portal.SSH.Master != tt.wantMaster {
				t.Errorf("portal doc SSH.Master: got %d, want %d", doc.Portal.SSH.Master, tt.wantMaster)
			}
			if doc.Portal.SSH.Authenticated {
				t.Errorf("portal doc SSH.Authenticated: got true, want false")
			}
			if doc.Portal.Kubeconfig != nil {
				t.Errorf("portal doc Portal.Kubeconfig should be nil, got %+v", doc.Portal.Kubeconfig)
			}
			if doc.TTL != 60 {
				t.Errorf("portal doc TTL: got %d seconds, want 60 (1m)", doc.TTL)
			}
		})
	}
}
