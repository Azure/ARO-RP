package holmes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"
	"strconv"
)

// HolmesConfig holds configuration for HolmesGPT investigation pods.
type HolmesConfig struct {
	Image                       string
	AzureAPIKey                 string
	AzureAPIBase                string
	AzureAPIVersion             string
	Model                       string
	DefaultTimeout              int
	MaxConcurrentInvestigations int
}

// NewHolmesConfigFromEnv loads Holmes configuration from environment variables.
func NewHolmesConfigFromEnv() *HolmesConfig {
	return &HolmesConfig{
		Image:                       envOrDefault("HOLMES_IMAGE", "quay.io/haoran/holmesgpt:latest"),
		AzureAPIKey:                 os.Getenv("HOLMES_AZURE_API_KEY"),
		AzureAPIBase:                os.Getenv("HOLMES_AZURE_API_BASE"),
		AzureAPIVersion:             envOrDefault("HOLMES_AZURE_API_VERSION", "2025-04-01-preview"),
		Model:                       envOrDefault("HOLMES_MODEL", "azure/gpt-5.2"),
		DefaultTimeout:              envOrDefaultInt("HOLMES_DEFAULT_TIMEOUT", 600),
		MaxConcurrentInvestigations: envOrDefaultInt("HOLMES_MAX_CONCURRENT", 20),
	}
}

// Validate checks that required configuration values are set.
func (c *HolmesConfig) Validate() error {
	if c.AzureAPIKey == "" {
		return fmt.Errorf("HOLMES_AZURE_API_KEY is required")
	}
	if c.AzureAPIBase == "" {
		return fmt.Errorf("HOLMES_AZURE_API_BASE is required")
	}
	if c.Image == "" {
		return fmt.Errorf("HOLMES_IMAGE is required")
	}
	return nil
}

func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func envOrDefaultInt(key string, defaultValue int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return i
}
