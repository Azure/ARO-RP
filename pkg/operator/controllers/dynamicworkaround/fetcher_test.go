package dynamicworkaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"

	mock_azsecrets "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azsecrets"
)

// goodCatalogJSON returns a JSON body that passes Validate. Tests can mutate
// it before sending to exercise individual validation failure paths.
func goodCatalogJSON(t *testing.T) []byte {
	t.Helper()
	c := Catalog{
		SchemaVersion:  SchemaVersion,
		CatalogVersion: "fetcher-test-1",
		Workarounds: []Workaround{
			{
				Name:              "test-wa",
				MachineConfigName: "99-test-wa",
				Role:              "worker",
				Ignition:          json.RawMessage(`{"ignition":{"version":"3.2.0"}}`),
			},
		},
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// secretResp wraps a raw string value into the exact response shape the real
// Key Vault SDK returns. Keeping the helper here means individual tests don't
// have to know that .Value is a *string.
func secretResp(value string) azsecrets.GetSecretResponse {
	return azsecrets.GetSecretResponse{
		Secret: azsecrets.Secret{Value: &value},
	}
}

func TestKeyVaultFetcherHappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mc := mock_azsecrets.NewMockClient(ctrl)
	mc.EXPECT().
		GetSecret(gomock.Any(), "my-catalog", "", gomock.Nil()).
		Return(secretResp(string(goodCatalogJSON(t))), nil)

	f := NewKeyVaultFetcher()
	cat, err := f.Fetch(context.Background(), mc, "my-catalog", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cat.CatalogVersion != "fetcher-test-1" {
		t.Errorf("catalogVersion = %q, want fetcher-test-1", cat.CatalogVersion)
	}
	if len(cat.Workarounds) != 1 {
		t.Errorf("workarounds = %d, want 1", len(cat.Workarounds))
	}
}

func TestKeyVaultFetcherWithVersion(t *testing.T) {
	// Verify the version is threaded through to the SDK call so pinning works.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mc := mock_azsecrets.NewMockClient(ctrl)
	mc.EXPECT().
		GetSecret(gomock.Any(), "my-catalog", "abc123", gomock.Nil()).
		Return(secretResp(string(goodCatalogJSON(t))), nil)

	f := NewKeyVaultFetcher()
	if _, err := f.Fetch(context.Background(), mc, "my-catalog", "abc123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKeyVaultFetcherSDKError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mc := mock_azsecrets.NewMockClient(ctrl)
	mc.EXPECT().
		GetSecret(gomock.Any(), "missing", "", gomock.Nil()).
		Return(azsecrets.GetSecretResponse{}, errors.New("secret not found"))

	f := NewKeyVaultFetcher()
	_, err := f.Fetch(context.Background(), mc, "missing", "")
	if err == nil {
		t.Fatal("expected error from SDK to propagate")
	}
	if !strings.Contains(err.Error(), "get secret") {
		t.Errorf("error %q does not mention get secret", err.Error())
	}
}

func TestKeyVaultFetcherEmptyValue(t *testing.T) {
	// Key Vault should never return a secret with nil Value, but if it does
	// we must fail loudly rather than apply an empty catalog (which would
	// tear down all managed MachineConfigs on the next reconcile).
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mc := mock_azsecrets.NewMockClient(ctrl)
	mc.EXPECT().
		GetSecret(gomock.Any(), "empty", "", gomock.Nil()).
		Return(azsecrets.GetSecretResponse{}, nil)

	f := NewKeyVaultFetcher()
	_, err := f.Fetch(context.Background(), mc, "empty", "")
	if err == nil || !strings.Contains(err.Error(), "empty value") {
		t.Fatalf("expected empty value error, got %v", err)
	}
}

func TestKeyVaultFetcherBodyTooLarge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	huge := strings.Repeat("x", int(MaxCatalogBytes)+10)
	mc := mock_azsecrets.NewMockClient(ctrl)
	mc.EXPECT().
		GetSecret(gomock.Any(), "huge", "", gomock.Nil()).
		Return(secretResp(huge), nil)

	f := NewKeyVaultFetcher()
	_, err := f.Fetch(context.Background(), mc, "huge", "")
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected size cap error, got %v", err)
	}
}

func TestKeyVaultFetcherInvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mc := mock_azsecrets.NewMockClient(ctrl)
	mc.EXPECT().
		GetSecret(gomock.Any(), "junk", "", gomock.Nil()).
		Return(secretResp("definitely not json"), nil)

	f := NewKeyVaultFetcher()
	_, err := f.Fetch(context.Background(), mc, "junk", "")
	if err == nil || !strings.Contains(err.Error(), "parse catalog JSON") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestKeyVaultFetcherValidationFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mc := mock_azsecrets.NewMockClient(ctrl)
	mc.EXPECT().
		GetSecret(gomock.Any(), "bad-schema", "", gomock.Nil()).
		Return(secretResp(`{"schemaVersion":"v999","catalogVersion":"x","workarounds":[]}`), nil)

	f := NewKeyVaultFetcher()
	_, err := f.Fetch(context.Background(), mc, "bad-schema", "")
	if err == nil || !strings.Contains(err.Error(), "invalid catalog") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestParseSecretURI(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		wantVault   string
		wantName    string
		wantVersion string
		wantErr     string
	}{
		{
			name:      "name only",
			uri:       "https://my-vault.vault.azure.net/secrets/cat",
			wantVault: "https://my-vault.vault.azure.net",
			wantName:  "cat",
		},
		{
			name:        "name + version",
			uri:         "https://my-vault.vault.azure.net/secrets/cat/abc123",
			wantVault:   "https://my-vault.vault.azure.net",
			wantName:    "cat",
			wantVersion: "abc123",
		},
		{
			name:      "trailing slash on path-only form",
			uri:       "https://my-vault.vault.azure.net/secrets/cat/",
			wantVault: "https://my-vault.vault.azure.net",
			wantName:  "cat",
		},
		{
			name:    "http scheme rejected",
			uri:     "http://my-vault.vault.azure.net/secrets/cat",
			wantErr: "https",
		},
		{
			name:    "missing host",
			uri:     "https:///secrets/cat",
			wantErr: "missing host",
		},
		{
			name:    "non-secrets path",
			uri:     "https://my-vault.vault.azure.net/keys/cat",
			wantErr: "/secrets/",
		},
		{
			name:    "missing name",
			uri:     "https://my-vault.vault.azure.net/secrets/",
			wantErr: "/secrets/",
		},
		{
			name:    "extra path segment",
			uri:     "https://my-vault.vault.azure.net/secrets/cat/abc/extra",
			wantErr: "trailing path",
		},
		{
			name:    "garbage",
			uri:     "://not a url",
			wantErr: "invalid secret URI",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vault, name, version, err := parseSecretURI(tt.uri)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
			if tt.wantErr != "" {
				return
			}
			if vault != tt.wantVault {
				t.Errorf("vault = %q, want %q", vault, tt.wantVault)
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if version != tt.wantVersion {
				t.Errorf("version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}
