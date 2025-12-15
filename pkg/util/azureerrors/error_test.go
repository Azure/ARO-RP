package azureerrors

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
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
					Code:    CODE_AUTHFAILED,
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
			name: "Non-deploy authorization Failed",
			err: autorest.DetailedError{
				Original: &azure.RequestError{
					DetailedError: autorest.DetailedError{
						StatusCode: "403",
					},
					ServiceError: &azure.ServiceError{
						Code:    CODE_AUTHFAILED,
						Message: "The client 'c78f37e4-e979-4b70-8055-d04f6e6c0302' with object id 'c78f37e4-e979-4b70-8055-d04f6e6c0302' does not have authorization to perform action 'Microsoft.Network/virtualNetworks/read' over scope '/subscriptions/46626fc5-476d-41ad-8c76-2ec49c6994eb/resourceGroups/v4-e2e-V36907046-centralus/providers/Microsoft.Network/virtualNetworks/dev-vnet' or the scope is invalid. If access was recently granted, please refresh your credentials.",
					},
				},
				PackageType: "network.VirtualNetworksClient",
				Method:      "Get",
				StatusCode:  http.StatusForbidden,
				Message:     "Failure responding to request",
				// Response omitted for brevity
			},
			want: true,
		},
		{
			name: "Nested authorization failed",
			err: &azure.ServiceError{
				Code:    CODE_DEPLOYFAILED,
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
		{
			name: "azcore ResponseError with AuthorizationFailed",
			err: &azcore.ResponseError{
				StatusCode: http.StatusForbidden,
				ErrorCode:  CODE_AUTHFAILED,
				RawResponse: &http.Response{
					StatusCode: http.StatusForbidden,
				},
			},
			want: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := HasAuthorizationFailedError(tt.err)
			if got != tt.want {
				t.Error(got)
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
						Code:    CODE_DEPLOYACTIVE,
						Message: "Unable to edit or replace deployment 'deployment': previous deployment from '4/4/2020 2:17:07 AM' is still active (expiration time is '4/11/2020 2:17:01 AM'). Please see https://aka.ms/arm-deploy for usage details.",
					},
				},
				PackageType: "resources.DeploymentsClient",
				Method:      "CreateOrUpdate",
				Message:     "Failure sending request",
			},
			want: true,
		},
		{
			name: "azcore ResponseError with DeploymentActive",
			err: &azcore.ResponseError{
				ErrorCode: CODE_DEPLOYACTIVE,
			},
			want: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDeploymentActiveError(autorest.NewErrorWithError(tt.err, "", "", nil, ""))
			if got != tt.want {
				t.Error(got)
			}
		})
	}
}

func TestIsDeploymentMissingPermissionsError(t *testing.T) {
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
			name: "Missing RoleAssignment",
			err: autorest.DetailedError{
				PackageType: "features.DeploymentsClient",
				Method:      "CreateOrUpdate",
				Message:     "Failure sending request",
				Original: &azure.ServiceError{
					Code:    CODE_INVALIDTEMPL,
					Message: "The template deployment failed with error: 'Authorization failed for template resource '$RESOURCE' of type 'Microsoft.Authorization/roleAssignments'. The client '$CLIENT' with object id '$CLIENT' does not have permission to perform action '$ACTION' at scope '$SCOPE'.'.",
				},
			},
			want: true,
		},
		{
			name: "azcore ResponseError with InvalidTemplateDeployment",
			err: &azcore.ResponseError{
				ErrorCode: CODE_INVALIDTEMPL,
			},
			// We want false because it's missing the "Authorization failed for
			// template resource" message.
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDeploymentMissingPermissionsError(autorest.NewErrorWithError(tt.err, "", "", nil, ""))
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsVMSKUError tests detection of VM SKU availability errors.
// Azure Resource Manager error codes: https://learn.microsoft.com/en-us/azure/azure-resource-manager/troubleshooting/error-sku-not-available
// ARO RP validation uses InvalidParameter with SKU in the message.
func TestIsVMSKUError(t *testing.T) {
	for _, tt := range []struct {
		name            string
		err             error
		wantIsVMError   bool
		wantProfileType VMProfileType
	}{
		{
			name:            "nil error",
			err:             nil,
			wantIsVMError:   false,
			wantProfileType: VMProfileUnknown,
		},
		{
			name:            "non-VM error",
			err:             errors.New("some random error"),
			wantIsVMError:   false,
			wantProfileType: VMProfileUnknown,
		},
		{
			name:            "InvalidParameter without SKU",
			err:             errors.New("Code=\"InvalidParameter\" Message=\"Some other parameter issue\""),
			wantIsVMError:   false,
			wantProfileType: VMProfileUnknown,
		},
		{
			name:            "InvalidParameter SKU error - worker profile",
			err:             errors.New("Code=\"InvalidParameter\" Message=\"The selected SKU 'Standard_D4s_v5' is restricted\" Target=\"properties.workerProfiles[0].VMSize\""),
			wantIsVMError:   true,
			wantProfileType: VMProfileWorker,
		},
		{
			name:            "InvalidParameter SKU error - master profile lowercase",
			err:             errors.New("Code=\"InvalidParameter\" Message=\"The selected SKU 'Standard_D8s_v5' is restricted\" Target=\"properties.masterProfile.VMSize\""),
			wantIsVMError:   true,
			wantProfileType: VMProfileMaster,
		},
		{
			name:            "InvalidParameter SKU error - master profile uppercase",
			err:             errors.New("Code=\"InvalidParameter\" Message=\"The selected SKU 'Standard_D8s_v5' is restricted\" Target=\"properties.MasterProfile.VMSize\""),
			wantIsVMError:   true,
			wantProfileType: VMProfileMaster,
		},
		{
			name:            "InvalidParameter SKU error - no profile info",
			err:             errors.New("Code=\"InvalidParameter\" Message=\"The selected SKU is restricted in this region\""),
			wantIsVMError:   true,
			wantProfileType: VMProfileUnknown,
		},
		{
			name:            "SkuNotAvailable error",
			err:             errors.New("Code=\"SkuNotAvailable\" Message=\"The requested size for resource is currently not available\""),
			wantIsVMError:   true,
			wantProfileType: VMProfileUnknown,
		},
		{
			name:            "SkuNotAvailable error - worker profile",
			err:             errors.New("Code=\"SkuNotAvailable\" Target=\"properties.workerProfiles[0].VMSize\""),
			wantIsVMError:   true,
			wantProfileType: VMProfileWorker,
		},
		{
			name:            "NotAvailableForSubscription error",
			err:             errors.New("Restrictions: NotAvailableForSubscription, type: Zone, locations: westeurope"),
			wantIsVMError:   true,
			wantProfileType: VMProfileUnknown,
		},
		{
			name:            "size not available in location",
			err:             errors.New("The requested size for resource is currently not available in location 'westeurope'"),
			wantIsVMError:   true,
			wantProfileType: VMProfileUnknown,
		},
		{
			name:            "generic deployment error - not a SKU error",
			err:             errors.New("Code=\"DeploymentFailed\" Message=\"Deployment failed\""),
			wantIsVMError:   false,
			wantProfileType: VMProfileUnknown,
		},
		{
			name:            "authorization error - not a SKU error",
			err:             errors.New("Code=\"AuthorizationFailed\" Message=\"The client does not have authorization\""),
			wantIsVMError:   false,
			wantProfileType: VMProfileUnknown,
		},
		{
			name: "azcore ResponseError with SkuNotAvailable",
			err: &azcore.ResponseError{
				ErrorCode: CODE_SKUNOTAVAILABLE,
			},
			wantIsVMError:   true,
			wantProfileType: VMProfileUnknown,
		},
		{
			name: "autorest DetailedError with SkuNotAvailable ServiceError",
			err: autorest.DetailedError{
				Original: &azure.ServiceError{
					Code:    CODE_SKUNOTAVAILABLE,
					Message: "The requested size for resource is currently not available in location 'eastus'",
				},
			},
			wantIsVMError:   true,
			wantProfileType: VMProfileUnknown,
		},
		{
			name: "autorest DetailedError with InvalidParameter SKU ServiceError - worker",
			err: autorest.DetailedError{
				Original: &azure.ServiceError{
					Code:    CODE_INVALIDPARAM,
					Message: "The selected SKU 'Standard_D4s_v5' is restricted for workerProfiles[0].VMSize",
				},
			},
			wantIsVMError:   true,
			wantProfileType: VMProfileWorker,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gotIsVMError, gotProfileType := IsVMSKUError(tt.err)

			if gotIsVMError != tt.wantIsVMError {
				t.Errorf("IsVMSKUError() isVMError = %v, want %v", gotIsVMError, tt.wantIsVMError)
			}
			if gotProfileType != tt.wantProfileType {
				t.Errorf("IsVMSKUError() profileType = %v, want %v", gotProfileType, tt.wantProfileType)
			}
		})
	}
}
