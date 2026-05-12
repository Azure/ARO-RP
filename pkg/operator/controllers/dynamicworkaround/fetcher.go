package dynamicworkaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
)

// Fetcher loads and validates a catalog from an Azure Key Vault secret.
//
// The Fetcher is deliberately Azure-aware but credential-agnostic: callers
// build the azsecrets.Client (with whatever credential is appropriate) and
// hand it in alongside the secret name/version. This keeps the fetcher itself
// pure logic (read → validate) and makes it trivial to mock in tests using
// the existing mock_azsecrets.MockClient.
type Fetcher interface {
	Fetch(ctx context.Context, client azsecrets.Client, name, version string) (*Catalog, error)
}

// NewKeyVaultFetcher returns the production fetcher. The receiver carries no
// state today; the constructor exists so tests can swap in stub fetchers and
// callers don't have to import the concrete type.
func NewKeyVaultFetcher() Fetcher {
	return &keyVaultFetcher{}
}

type keyVaultFetcher struct{}

// Fetch retrieves the secret value, enforces a body size cap, and validates
// the resulting catalog. Any failure (network, schema, validation) is wrapped
// so the caller can log + skip the reconcile cycle cleanly.
//
// version may be empty, in which case Key Vault returns the latest version.
func (f *keyVaultFetcher) Fetch(ctx context.Context, client azsecrets.Client, name, version string) (*Catalog, error) {
	resp, err := client.GetSecret(ctx, name, version, nil)
	if err != nil {
		return nil, fmt.Errorf("get secret %q: %w", name, err)
	}
	if resp.Value == nil {
		return nil, fmt.Errorf("secret %q returned an empty value", name)
	}

	// The Key Vault API caps secret values at 25 KiB today, but we apply our
	// own cap as a second line of defence in case the SDK or service ever
	// raises that limit — a runaway secret should not OOM the operator.
	body := []byte(*resp.Value)
	if int64(len(body)) > MaxCatalogBytes {
		return nil, fmt.Errorf("secret value exceeds %d bytes", MaxCatalogBytes)
	}

	cat := &Catalog{}
	if err := json.Unmarshal(body, cat); err != nil {
		return nil, fmt.Errorf("parse catalog JSON: %w", err)
	}
	if err := cat.Validate(); err != nil {
		return nil, fmt.Errorf("invalid catalog: %w", err)
	}
	return cat, nil
}

// parseSecretURI splits a Key Vault secret URI into its (vault URL, name,
// version) parts. Both shapes are accepted:
//
//	https://<vault>.vault.azure.net/secrets/<name>
//	https://<vault>.vault.azure.net/secrets/<name>/<version>
//
// The version is the empty string in the first form, which is the value the
// azsecrets SDK uses to mean "latest".
//
// We intentionally enforce the https scheme here (rather than at the fetcher)
// so the controller can fail fast and log a clear error before reaching for
// credentials.
func parseSecretURI(raw string) (vaultURL, name, version string, err error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid secret URI: %w", err)
	}
	if u.Scheme != "https" {
		return "", "", "", fmt.Errorf("secret URI must be https (got scheme %q)", u.Scheme)
	}
	if u.Host == "" {
		return "", "", "", fmt.Errorf("secret URI is missing host")
	}

	// Path shape: /secrets/<name> or /secrets/<name>/<version>
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "secrets" || parts[1] == "" {
		return "", "", "", fmt.Errorf("secret URI path must be /secrets/<name>[/<version>], got %q", u.Path)
	}
	name = parts[1]
	if len(parts) >= 3 {
		version = parts[2]
	}
	if len(parts) > 3 {
		return "", "", "", fmt.Errorf("secret URI has unexpected trailing path: %q", u.Path)
	}

	// Reconstruct the canonical vault URL (scheme + host) and drop the path —
	// azsecrets.NewClient expects only the vault endpoint, not the secret URL.
	vaultURL = (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
	return vaultURL, name, version, nil
}
