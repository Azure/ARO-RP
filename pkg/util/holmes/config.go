package holmes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// modelPattern validates the model name contains only safe characters
// (alphanumeric, slashes, dots, colons, hyphens, underscores).
var modelPattern = regexp.MustCompile(`^[a-zA-Z0-9/.:_-]+$`)

const (
	holmesAzureAPIBaseSecretName = "holmes-azure-api-base"
)

// HolmesConfig holds configuration for HolmesGPT investigation pods.
type HolmesConfig struct {
	Image                       string
	AzureAPIBase                string
	AzureAPIVersion             string
	Model                       string
	DefaultTimeout              int
	MaxConcurrentInvestigations int
	UAMIClientID                string
}

// NewHolmesConfigFromEnv loads all config from environment variables.
// Used in local development mode (RP_MODE=development).
func NewHolmesConfigFromEnv(acrDomain string) (*HolmesConfig, error) {
	c, err := newHolmesConfigBase(acrDomain)
	if err != nil {
		return nil, err
	}
	c.AzureAPIBase = os.Getenv("HOLMES_AZURE_API_BASE")
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// NewHolmesConfig loads non-secret config from env vars and secrets from Key Vault.
// Used in production mode.
func NewHolmesConfig(ctx context.Context, acrDomain string, serviceKeyvault azsecrets.Client) (*HolmesConfig, error) {
	apiBaseResp, err := serviceKeyvault.GetSecret(ctx, holmesAzureAPIBaseSecretName, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s from keyvault: %w", holmesAzureAPIBaseSecretName, err)
	}
	if apiBaseResp.Value == nil {
		return nil, fmt.Errorf("keyvault secret %s has no value", holmesAzureAPIBaseSecretName)
	}

	c, err := newHolmesConfigBase(acrDomain)
	if err != nil {
		return nil, err
	}
	c.AzureAPIBase = *apiBaseResp.Value
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// newHolmesConfigBase loads the non-secret configuration from environment variables.
// The acrDomain is used to construct the default Holmes image pullspec.
func newHolmesConfigBase(acrDomain string) (*HolmesConfig, error) {
	defaultTimeout, err := envOrDefaultInt("HOLMES_DEFAULT_TIMEOUT", 600)
	if err != nil {
		return nil, err
	}
	maxConcurrent, err := envOrDefaultInt("HOLMES_MAX_CONCURRENT", 20)
	if err != nil {
		return nil, err
	}
	return &HolmesConfig{
		Image:                       envOrDefault("HOLMES_IMAGE", version.HolmesImage(acrDomain)),
		AzureAPIVersion:             envOrDefault("HOLMES_AZURE_API_VERSION", "2025-04-01-preview"),
		Model:                       envOrDefault("HOLMES_MODEL", "azure/gpt-5.2"),
		DefaultTimeout:              defaultTimeout,
		MaxConcurrentInvestigations: maxConcurrent,
		UAMIClientID:                os.Getenv("HOLMES_UAMI_CLIENT_ID"),
	}, nil
}

// Validate checks that required configuration values are set.
func (c *HolmesConfig) Validate() error {
	if c.UAMIClientID == "" {
		return fmt.Errorf("holmes UAMI client ID is required (set HOLMES_UAMI_CLIENT_ID)")
	}
	if c.AzureAPIBase == "" {
		return fmt.Errorf("holmes Azure API base is required")
	}
	if c.Image == "" {
		return fmt.Errorf("holmes image is required")
	}
	if !modelPattern.MatchString(c.Model) {
		return fmt.Errorf("holmes model name contains invalid characters")
	}
	if c.DefaultTimeout <= 0 {
		return fmt.Errorf("holmes default timeout must be greater than 0")
	}
	if c.MaxConcurrentInvestigations <= 0 {
		return fmt.Errorf("holmes max concurrent investigations must be greater than 0")
	}
	return nil
}

// NewHolmesConfigForTest creates a HolmesConfig with all fields set directly,
// bypassing env vars and Key Vault. Intended for use in tests only.
func NewHolmesConfigForTest(image, apiBase, apiVersion, model string, timeout, maxConcurrent int) *HolmesConfig {
	return &HolmesConfig{
		Image:                       image,
		AzureAPIBase:                apiBase,
		AzureAPIVersion:             apiVersion,
		Model:                       model,
		DefaultTimeout:              timeout,
		MaxConcurrentInvestigations: maxConcurrent,
		UAMIClientID:                "test-uami-client-id",
	}
}

func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func envOrDefaultInt(key string, defaultValue int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue, nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value for %s: %w", key, err)
	}
	return i, nil
}
