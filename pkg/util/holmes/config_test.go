package holmes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"

	mock_azsecrets "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azsecrets"
)

func TestNewHolmesConfigFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "valid config with all required env vars",
			envVars: map[string]string{
				"HOLMES_AZURE_API_KEY":  "test-key",
				"HOLMES_AZURE_API_BASE": "https://test.openai.azure.com",
			},
		},
		{
			name: "missing API key returns error",
			envVars: map[string]string{
				"HOLMES_AZURE_API_BASE": "https://test.openai.azure.com",
			},
			wantErr: true,
		},
		{
			name: "missing API base returns error",
			envVars: map[string]string{
				"HOLMES_AZURE_API_KEY": "test-key",
			},
			wantErr: true,
		},
		{
			name: "custom values override defaults",
			envVars: map[string]string{
				"HOLMES_AZURE_API_KEY":     "custom-key",
				"HOLMES_AZURE_API_BASE":    "https://custom.openai.azure.com",
				"HOLMES_IMAGE":             "custom-image:v1",
				"HOLMES_MODEL":             "azure/gpt-4o",
				"HOLMES_DEFAULT_TIMEOUT":   "300",
				"HOLMES_MAX_CONCURRENT":    "5",
				"HOLMES_AZURE_API_VERSION": "2024-01-01",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all Holmes env vars, then set test values.
			for _, key := range []string{
				"HOLMES_AZURE_API_KEY", "HOLMES_AZURE_API_BASE", "HOLMES_IMAGE",
				"HOLMES_MODEL", "HOLMES_DEFAULT_TIMEOUT", "HOLMES_MAX_CONCURRENT",
				"HOLMES_AZURE_API_VERSION",
			} {
				t.Setenv(key, "")
			}
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg, err := NewHolmesConfigFromEnv("arosvc.azurecr.io")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.envVars["HOLMES_AZURE_API_KEY"], cfg.AzureAPIKey)
			require.Equal(t, tt.envVars["HOLMES_AZURE_API_BASE"], cfg.AzureAPIBase)

			if tt.envVars["HOLMES_IMAGE"] != "" {
				require.Equal(t, tt.envVars["HOLMES_IMAGE"], cfg.Image)
			}
			if tt.envVars["HOLMES_MODEL"] != "" {
				require.Equal(t, tt.envVars["HOLMES_MODEL"], cfg.Model)
			}
			if tt.envVars["HOLMES_DEFAULT_TIMEOUT"] != "" {
				require.Equal(t, 300, cfg.DefaultTimeout)
			}
			if tt.envVars["HOLMES_MAX_CONCURRENT"] != "" {
				require.Equal(t, 5, cfg.MaxConcurrentInvestigations)
			}
		})
	}
}

func TestNewHolmesConfig(t *testing.T) {
	ctx := context.Background()

	apiKey := "keyvault-api-key"
	apiBase := "https://keyvault.openai.azure.com"

	tests := []struct {
		name    string
		mocks   func(*mock_azsecrets.MockClient)
		wantErr bool
	}{
		{
			name: "reads secrets from keyvault",
			mocks: func(m *mock_azsecrets.MockClient) {
				m.EXPECT().GetSecret(ctx, holmesAzureAPIKeySecretName, "", nil).
					Return(azsecrets.GetSecretResponse{
						Secret: azsecrets.Secret{Value: &apiKey},
					}, nil)
				m.EXPECT().GetSecret(ctx, holmesAzureAPIBaseSecretName, "", nil).
					Return(azsecrets.GetSecretResponse{
						Secret: azsecrets.Secret{Value: &apiBase},
					}, nil)
			},
		},
		{
			name: "API key not found in keyvault returns error",
			mocks: func(m *mock_azsecrets.MockClient) {
				m.EXPECT().GetSecret(ctx, holmesAzureAPIKeySecretName, "", nil).
					Return(azsecrets.GetSecretResponse{}, fmt.Errorf("secret not found"))
			},
			wantErr: true,
		},
		{
			name: "API base not found in keyvault returns error",
			mocks: func(m *mock_azsecrets.MockClient) {
				m.EXPECT().GetSecret(ctx, holmesAzureAPIKeySecretName, "", nil).
					Return(azsecrets.GetSecretResponse{
						Secret: azsecrets.Secret{Value: &apiKey},
					}, nil)
				m.EXPECT().GetSecret(ctx, holmesAzureAPIBaseSecretName, "", nil).
					Return(azsecrets.GetSecretResponse{}, fmt.Errorf("secret not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockKV := mock_azsecrets.NewMockClient(controller)
			tt.mocks(mockKV)

			cfg, err := NewHolmesConfig(ctx, "arosvc.azurecr.io", mockKV)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, apiKey, cfg.AzureAPIKey)
			require.Equal(t, apiBase, cfg.AzureAPIBase)
			// Non-secret values should still come from env/defaults
			require.NotEmpty(t, cfg.Image)
			require.NotEmpty(t, cfg.Model)
		})
	}
}
