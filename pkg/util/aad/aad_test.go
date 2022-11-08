package aad

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_refreshable "github.com/Azure/ARO-RP/pkg/util/mocks/refreshable"
)

func TestAuthenticateServicePrincipalToken(t *testing.T) {
	tests := []struct {
		name         string
		wantErr      error
		refreshError error
	}{
		{
			name:         "Calling ServicePrincipal returns error.",
			refreshError: errors.New("Some refesh error"),
			wantErr: api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidServicePrincipalCredentials,
				"properties.servicePrincipalProfile",
				"The provided service principal credentials are invalid."),
		},
	}

	mockController := gomock.NewController(t)
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := 1 * time.Second
			authorizer := mock_refreshable.NewMockAuthorizer(mockController)

			authorizer.EXPECT().RefreshWithContext(gomock.Any(), gomock.Any()).Return(false, tt.refreshError).AnyTimes()

			err := AuthenticateServicePrincipalToken(ctx, log, authorizer, duration)
			if err != nil && tt.wantErr.Error() != err.Error() {
				t.Errorf("AuthenticateServicePrincipalToken() error = %v, wantErr = %v", err.Error(), tt.wantErr.Error())
				return
			}
		})
	}
}
