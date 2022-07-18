package dynamic

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclaim"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestValidateServicePrincipal(t *testing.T) {
	// Build test environment
	controller := gomock.NewController(t)
	defer controller.Finish()
	_, logger := testlog.New()
	ctx := context.Background()
	testInterface, _ := env.NewEnv(ctx, logger)

	// Declare global variables needed for testing
	conf, _ := adal.NewOAuthConfig("endpoint", "test")
	jwtEncodingForTests := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1bml0dGVzdCI6InVuaXRUZXN0In0.82iv0dJ8E-3PFPRmD-a2yrwuDlMaILW_GhPETahW96w"
	serviceProvider, _ := adal.NewServicePrincipalTokenFromManualToken(
		*conf,
		"clientID",
		"resource",
		adal.Token{
			AccessToken: jwtEncodingForTests,
			Type:        "TEST",
			ExpiresIn:   "3600",
		})
	dyn, _ := NewValidator(logger,
		testInterface,
		&azureclient.AROEnvironment{},
		"",
		refreshable.NewAuthorizer(&adal.ServicePrincipalToken{}),
		AuthorizerFirstParty)

	// Declare and run ValidateServicePrincipal function test suite
	for _, tt := range []struct {
		name         string
		tokenWrapper func(ctx context.Context, l *logrus.Entry, clientID, clientSecret, tenantID, ActiveDirectoryEndpoint, GraphEndpoint string) (*adal.ServicePrincipalToken, error)
		azClaim      azureclaim.AzureClaim
		expectedErr  error
	}{
		{
			name:    "failed to retrieve token",
			azClaim: azureclaim.AzureClaim{},
			tokenWrapper: func(ctx context.Context, l *logrus.Entry, clientID, clientSecret, tenantID, ActiveDirectoryEndpoint, GraphEndpoint string) (*adal.ServicePrincipalToken, error) {
				return serviceProvider, errors.New("failed to retrieve token")
			},
			expectedErr: errors.New("failed to retrieve token"),
		},
		{
			name:    "bad azure claim role",
			azClaim: azureclaim.AzureClaim{Roles: []string{"Application.ReadWrite.OwnedBy"}},
			tokenWrapper: func(ctx context.Context, l *logrus.Entry, clientID, clientSecret, tenantID, ActiveDirectoryEndpoint, GraphEndpoint string) (*adal.ServicePrincipalToken, error) {
				return serviceProvider, nil
			},
			expectedErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "The provided service principal must not have the Application.ReadWrite.OwnedBy permission."),
		},
		{
			name:    "success with non disruptive role",
			azClaim: azureclaim.AzureClaim{Roles: []string{"noneEmptyRole"}},
			tokenWrapper: func(ctx context.Context, l *logrus.Entry, clientID, clientSecret, tenantID, ActiveDirectoryEndpoint, GraphEndpoint string) (*adal.ServicePrincipalToken, error) {
				return serviceProvider, nil
			},
			expectedErr: nil,
		},
		{
			name:    "success with no roles",
			azClaim: azureclaim.AzureClaim{},
			tokenWrapper: func(ctx context.Context, l *logrus.Entry, clientID, clientSecret, tenantID, ActiveDirectoryEndpoint, GraphEndpoint string) (*adal.ServicePrincipalToken, error) {
				return serviceProvider, nil
			},
			expectedErr: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := dyn.ValidateServicePrincipal(ctx, "", "", "", tt.azClaim, tt.tokenWrapper)
			if err != nil {
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			}
		})
	}
}
