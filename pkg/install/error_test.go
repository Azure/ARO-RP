package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

func TestIsResourceQuotaExceededError(t *testing.T) {
	for _, tt := range []struct {
		name     string
		inputErr error
		wantBool bool
		wantMsg  string
	}{
		{
			name: "Another error",
			inputErr: autorest.NewErrorWithError(&azure.ServiceError{
				Details: []map[string]interface{}{{
					"code":    "AnotherCode",
					"message": "Something happened",
				}}}, "", "", nil, ""),
		},
		{
			name: "Quota exceeded",
			inputErr: autorest.NewErrorWithError(&azure.ServiceError{
				Details: []map[string]interface{}{{
					"code":    "QuotaExceeded",
					"message": "Quota exceeded",
				}}}, "", "", nil, ""),
			wantBool: true,
			wantMsg:  "Quota exceeded",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gotBool, gotMsg := isResourceQuotaExceededError(tt.inputErr)
			if gotBool != tt.wantBool {
				t.Error(gotBool)
			}
			if gotMsg != tt.wantMsg {
				t.Error(gotMsg)
			}
		})
	}
}

func TestIsDeploymentActiveError(t *testing.T) {
	for _, tt := range []struct {
		name     string
		inputErr error
		wantBool bool
	}{
		{
			name: "Another error",
			inputErr: autorest.NewErrorWithError(azure.RequestError{
				ServiceError: &azure.ServiceError{Code: "AnotherCode"},
			}, "", "", nil, ""),
		},
		{
			name: "Deployment active",
			inputErr: autorest.NewErrorWithError(azure.RequestError{
				ServiceError: &azure.ServiceError{Code: "DeploymentActive"},
			}, "", "", nil, ""),
			wantBool: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gotBool := isDeploymentActiveError(autorest.NewErrorWithError(tt.inputErr, "", "", nil, ""))
			if gotBool != tt.wantBool {
				t.Error(gotBool)
			}
		})
	}
}
