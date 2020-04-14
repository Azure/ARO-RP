package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

// The tests in this file contain verbatim copies of errors returned from Azure
// transcribed using github.com/shurcooL/go-goon.  Consider these errors
// immutable, but feel free to add additional examples.  Rationale: it is really
// easy to introduce regressions here.

func TestHasAuthorizationFailedError(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "Another error",
			err:  errors.New("something happened"),
		},
		{
			name: "Authorization Failed",
			err: autorest.DetailedError{
				Original: &azure.ServiceError{
					Code:    "AuthorizationFailed",
					Message: "The client 'a0f3c32d-647d-416c-8997-fb2463b1dcd5' with object id 'a0f3c32d-647d-416c-8997-fb2463b1dcd5' does not have authorization to perform action 'Microsoft.Resources/deployments/write' over scope '/subscriptions/447cf33b-a19b-42f7-ab5e-b0b6f7be7525/resourcegroups/jmintertest/providers/Microsoft.Resources/deployments/deployment' or the scope is invalid. If access was recently granted, please refresh your credentials.",
				},
				PackageType: "resources.DeploymentsClient",
				Method:      "CreateOrUpdate",
				StatusCode:  http.StatusForbidden,
				Message:     "Failure sending request",
				// Response omitted for brevity
			},
			want: true,
		},
		{
			name: "Nested authorization failed",
			err: &azure.ServiceError{
				Code:    "DeploymentFailed",
				Message: "At least one resource deployment operation failed. Please list deployment operations for details. Please see https://aka.ms/DeployOperations for usage details.",
				Details: []map[string]interface{}{
					{
						"code":    "Forbidden",
						"message": "{\r\n  \"error\": {\r\n    \"code\": \"AuthorizationFailed\",\r\n    \"message\": \"The client 'a0f3c32d-647d-416c-8997-fb2463b1dcd5' with object id 'a0f3c32d-647d-416c-8997-fb2463b1dcd5' does not have authorization to perform action 'Microsoft.Storage/storageAccounts/write' over scope '/subscriptions/225e02bc-43d0-43d1-a01a-17e584a4ef69/resourceGroups/test' or the scope is invalid. If access was recently granted, please refresh your credentials.\"\r\n  }\r\n}",
					},
				},
			},
			want: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := hasAuthorizationFailedError(tt.err)
			if got != tt.want {
				t.Error(got)
			}
		})
	}
}

func TestHasResourceQuotaExceededError(t *testing.T) {
	for _, tt := range []struct {
		name    string
		err     error
		want    bool
		wantMsg string
	}{
		{
			name: "Another error",
			err:  errors.New("something happened"),
		},
		{
			name: "Quota exceeded",
			err: autorest.DetailedError{
				Original: &azure.ServiceError{
					Code:    "InvalidTemplateDeployment",
					Message: "The template deployment 'deployment' is not valid according to the validation procedure. The tracking id is 'fbcb273b-d681-48b3-ab98-905d05fa8564'. See inner errors for details.",
					Details: []map[string]interface{}{
						{
							"message": "Operation could not be completed as it results in exceeding approved standardMSFamily Cores quota. Additional details - Deployment Model: Resource Manager, Location: eastus, Current Limit: 0, Current Usage: 0, Additional Required: 128, (Minimum) New Limit Required: 128. Submit a request for Quota increase at https://aka.ms/ProdportalCRP/?#create/Microsoft.Support/Parameters/%7B%22subId%22:%22225e02bc-43d0-43d1-a01a-17e584a4ef69%22,%22pesId%22:%2206bfd9d3-516b-d5c6-5802-169c800dec89%22,%22supportTopicId%22:%22e12e3d1d-7fa0-af33-c6d0-3c50df9658a3%22%7D by specifying parameters listed in the ‘Details’ section for deployment to succeed. Please read more about quota limits at https://docs.microsoft.com/en-us/azure/azure-supportability/per-vm-quota-requests.",
							"code":    "QuotaExceeded",
						},
					},
				},
				PackageType: "resources.DeploymentsClient",
				Method:      "CreateOrUpdate",
				StatusCode:  http.StatusBadRequest,
				Message:     "Failure sending request",
				// Response omitted for brevity
			},
			want:    true,
			wantMsg: "Operation could not be completed as it results in exceeding approved standardMSFamily Cores quota. Additional details - Deployment Model: Resource Manager, Location: eastus, Current Limit: 0, Current Usage: 0, Additional Required: 128, (Minimum) New Limit Required: 128. Submit a request for Quota increase at https://aka.ms/ProdportalCRP/?#create/Microsoft.Support/Parameters/%7B%22subId%22:%22225e02bc-43d0-43d1-a01a-17e584a4ef69%22,%22pesId%22:%2206bfd9d3-516b-d5c6-5802-169c800dec89%22,%22supportTopicId%22:%22e12e3d1d-7fa0-af33-c6d0-3c50df9658a3%22%7D by specifying parameters listed in the ‘Details’ section for deployment to succeed. Please read more about quota limits at https://docs.microsoft.com/en-us/azure/azure-supportability/per-vm-quota-requests.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, gotMsg := hasResourceQuotaExceededError(tt.err)
			if got != tt.want {
				t.Error(got)
			}
			if gotMsg != tt.wantMsg {
				t.Error(gotMsg)
			}
		})
	}
}

func TestIsDeploymentActiveError(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "Another error",
			err:  errors.New("something happened"),
		},
		{
			name: "Deployment active",
			err: autorest.DetailedError{
				Original: azure.RequestError{
					ServiceError: &azure.ServiceError{
						Code:    "DeploymentActive",
						Message: "Unable to edit or replace deployment 'deployment': previous deployment from '4/4/2020 2:17:07 AM' is still active (expiration time is '4/11/2020 2:17:01 AM'). Please see https://aka.ms/arm-deploy for usage details.",
					},
				},
				PackageType: "resources.DeploymentsClient",
				Method:      "CreateOrUpdate",
				Message:     "Failure sending request",
			},
			want: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := isDeploymentActiveError(autorest.NewErrorWithError(tt.err, "", "", nil, ""))
			if got != tt.want {
				t.Error(got)
			}
		})
	}
}
