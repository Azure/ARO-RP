package msi

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	sdkazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi/fake"

	utilmsi "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armmsi"
)

func NewTestFederatedIdentityCredentialsClient(subscriptionID string) (*utilmsi.ArmFederatedIdentityCredentialsClient, error) {
	client, err := utilmsi.NewFederatedIdentityCredentialsClient(subscriptionID, &azfake.TokenCredential{}, &arm.ClientOptions{
		ClientOptions: sdkazcore.ClientOptions{
			Transport: newFakeFederatedIdentityCredentialsClient(),
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create FederatedIdentityCredentialsClient: %v", err)
	}

	return client, nil
}

func newFakeFederatedIdentityCredentialsClient() *fake.FederatedIdentityCredentialsServerTransport {
	fakeServer := &fake.FederatedIdentityCredentialsServer{
		CreateOrUpdate: initializeCreateAndUpdate(),
		Delete:         initializeDelete(),
		Get:            initializeGet(),
	}

	return fake.NewFederatedIdentityCredentialsServerTransport(fakeServer)
}

func initializeCreateAndUpdate() func(ctx context.Context, resourceGroupName string, resourceName string, federatedIdentityCredentialResourceName string, parameters armmsi.FederatedIdentityCredential, options *armmsi.FederatedIdentityCredentialsClientCreateOrUpdateOptions) (resp azfake.Responder[armmsi.FederatedIdentityCredentialsClientCreateOrUpdateResponse], errResp azfake.ErrorResponder) {
	return func(ctx context.Context, resourceGroupName string, resourceName string, federatedIdentityCredentialResourceName string, parameters armmsi.FederatedIdentityCredential, options *armmsi.FederatedIdentityCredentialsClientCreateOrUpdateOptions) (resp azfake.Responder[armmsi.FederatedIdentityCredentialsClientCreateOrUpdateResponse], errResp azfake.ErrorResponder) {
		response := armmsi.FederatedIdentityCredentialsClientCreateOrUpdateResponse{
			FederatedIdentityCredential: armmsi.FederatedIdentityCredential{
				Name:       to.Ptr(federatedIdentityCredentialResourceName),
				Type:       to.Ptr("Microsoft.ManagedIdentity/userAssignedIdentities/federatedIdentityCredentials"),
				ID:         to.Ptr(fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/identityName/federatedIdentityCredentials/%s", resourceGroupName, federatedIdentityCredentialResourceName)),
				Properties: parameters.Properties,
			},
		}
		resp.SetResponse(http.StatusOK, response, nil)
		return
	}
}

func initializeDelete() func(ctx context.Context, resourceGroupName string, resourceName string, federatedIdentityCredentialResourceName string, options *armmsi.FederatedIdentityCredentialsClientDeleteOptions) (resp azfake.Responder[armmsi.FederatedIdentityCredentialsClientDeleteResponse], errResp azfake.ErrorResponder) {
	return func(ctx context.Context, resourceGroupName string, resourceName string, federatedIdentityCredentialResourceName string, options *armmsi.FederatedIdentityCredentialsClientDeleteOptions) (resp azfake.Responder[armmsi.FederatedIdentityCredentialsClientDeleteResponse], errResp azfake.ErrorResponder) {
		response := armmsi.FederatedIdentityCredentialsClientDeleteResponse{}
		resp.SetResponse(http.StatusOK, response, nil)
		return
	}
}

func initializeGet() func(ctx context.Context, resourceGroupName string, resourceName string, federatedIdentityCredentialResourceName string, options *armmsi.FederatedIdentityCredentialsClientGetOptions) (resp azfake.Responder[armmsi.FederatedIdentityCredentialsClientGetResponse], errResp azfake.ErrorResponder) {
	return func(ctx context.Context, resourceGroupName string, resourceName string, federatedIdentityCredentialResourceName string, options *armmsi.FederatedIdentityCredentialsClientGetOptions) (resp azfake.Responder[armmsi.FederatedIdentityCredentialsClientGetResponse], errResp azfake.ErrorResponder) {
		response := armmsi.FederatedIdentityCredentialsClientGetResponse{
			FederatedIdentityCredential: armmsi.FederatedIdentityCredential{
				Name:       to.Ptr(federatedIdentityCredentialResourceName),
				Type:       to.Ptr("Microsoft.ManagedIdentity/userAssignedIdentities/federatedIdentityCredentials"),
				ID:         to.Ptr(fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/identityName/federatedIdentityCredentials/%s", resourceGroupName, federatedIdentityCredentialResourceName)),
				Properties: &armmsi.FederatedIdentityCredentialProperties{},
			},
		}
		resp.SetResponse(http.StatusOK, response, nil)
		return
	}
}
