package holmes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"

	mock_azcore "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azcore"
	mock_azsecrets "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azsecrets"
)

func TestNewHolmesConfigFromEnv(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()
	mockCred := mock_azcore.NewMockTokenCredential(controller)

	tests := []struct {
		name    string
		envVars map[string]string
		cred    azcore.TokenCredential
		wantErr bool
	}{
		{
			name: "valid config with all required env vars",
			envVars: map[string]string{
				"HOLMES_AZURE_API_BASE": "https://test.openai.azure.com",
			},
			cred: mockCred,
		},
		{
			name:    "missing API base returns error",
			envVars: map[string]string{},
			cred:    mockCred,
			wantErr: true,
		},
		{
			name: "nil credential returns error",
			envVars: map[string]string{
				"HOLMES_AZURE_API_BASE": "https://test.openai.azure.com",
			},
			cred:    nil,
			wantErr: true,
		},
		{
			name: "custom values override defaults",
			envVars: map[string]string{
				"HOLMES_AZURE_API_BASE":    "https://custom.openai.azure.com",
				"HOLMES_IMAGE":             "custom-image:v1",
				"HOLMES_MODEL":             "azure/gpt-4o",
				"HOLMES_DEFAULT_TIMEOUT":   "300",
				"HOLMES_MAX_CONCURRENT":    "5",
				"HOLMES_AZURE_API_VERSION": "2024-01-01",
			},
			cred: mockCred,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{
				"HOLMES_AZURE_API_BASE", "HOLMES_IMAGE",
				"HOLMES_MODEL", "HOLMES_DEFAULT_TIMEOUT", "HOLMES_MAX_CONCURRENT",
				"HOLMES_AZURE_API_VERSION",
			} {
				t.Setenv(key, "")
			}
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg, err := NewHolmesConfigFromEnv("arosvc.azurecr.io", tt.cred)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
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

	apiBase := "https://keyvault.openai.azure.com"

	tests := []struct {
		name    string
		mocks   func(*mock_azsecrets.MockClient)
		wantErr bool
	}{
		{
			name: "reads API base from keyvault",
			mocks: func(m *mock_azsecrets.MockClient) {
				m.EXPECT().GetSecret(ctx, holmesAzureAPIBaseSecretName, "", nil).
					Return(azsecrets.GetSecretResponse{
						Secret: azsecrets.Secret{Value: &apiBase},
					}, nil)
			},
		},
		{
			name: "API base not found in keyvault returns error",
			mocks: func(m *mock_azsecrets.MockClient) {
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

			mockCred := mock_azcore.NewMockTokenCredential(controller)

			cfg, err := NewHolmesConfig(ctx, "arosvc.azurecr.io", mockKV, mockCred)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, apiBase, cfg.AzureAPIBase)
			require.NotEmpty(t, cfg.Image)
			require.NotEmpty(t, cfg.Model)
		})
	}
}

func TestAcquireToken(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		setupMock func(*mock_azcore.MockTokenCredential)
		wantToken string
		wantErr   bool
	}{
		{
			name: "successfully acquires token",
			setupMock: func(m *mock_azcore.MockTokenCredential) {
				m.EXPECT().GetToken(ctx, policy.TokenRequestOptions{
					Scopes: []string{cognitiveServicesScope},
				}).Return(azcore.AccessToken{
					Token:     "test-entra-token",
					ExpiresOn: time.Now().Add(time.Hour),
				}, nil)
			},
			wantToken: "test-entra-token",
		},
		{
			name: "token acquisition failure returns error",
			setupMock: func(m *mock_azcore.MockTokenCredential) {
				m.EXPECT().GetToken(ctx, policy.TokenRequestOptions{
					Scopes: []string{cognitiveServicesScope},
				}).Return(azcore.AccessToken{}, fmt.Errorf("credential expired"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockCred := mock_azcore.NewMockTokenCredential(controller)
			tt.setupMock(mockCred)

			cfg := &HolmesConfig{
				tokenCredential: mockCred,
			}

			token, err := cfg.AcquireToken(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantToken, token)
		})
	}
}
