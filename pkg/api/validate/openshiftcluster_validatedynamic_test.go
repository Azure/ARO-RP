package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mockaad "github.com/Azure/ARO-RP/pkg/util/mocks/aad"
	mockfeatures "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mocknetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	mock_instancemetadata "github.com/Azure/ARO-RP/pkg/util/mocks/instancemetadata"
)

func TestValidateServicePrincipalProfile(t *testing.T) {
	ctx := context.Background()

	// Valid JWT
	altsecidJWT := "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJUZXN0VmFsaWRhdGVTZXJ2aWNlUHJpbmNpcGFsUHJvZmlsZSIsImlhdCI6MTU4ODIxNzI1NiwiZXhwIjoxNjE5NzUzMjU2LCJhdWQiOiJ3d3cuZXhhbXBsZS5jb20iLCJzdWIiOiJ0ZXN0QGV4YW1wbGUuY29tIiwiYWx0c2VjaWQiOiJvayJ9.P4ETdlihD2YNGB9b4ARYX7IIEudP4f7a2xHcNCMzER8"

	// Invalid JWT: has Application.ReadWrite.OwnedBy permission
	// {
	// 	"iss": "TestValidateServicePrincipalProfile",
	// 	"iat": 1588217256,
	// 	"exp": 1619753256,
	// 	"aud": "www.example.com",
	// 	"sub": "test@example.com",
	// 	"altsecid": "ok",
	// 	"Roles": [
	// 		"Application.ReadWrite.OwnedBy",
	// 		"Test"
	// 	]
	// }
	invalidJWT := "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJUZXN0VmFsaWRhdGVTZXJ2aWNlUHJpbmNpcGFsUHJvZmlsZSIsImlhdCI6MTU4ODIxNzI1NiwiZXhwIjoxNjE5NzUzMjU2LCJhdWQiOiJ3d3cuZXhhbXBsZS5jb20iLCJzdWIiOiJ0ZXN0QGV4YW1wbGUuY29tIiwiYWx0c2VjaWQiOiJvayIsIlJvbGVzIjpbIkFwcGxpY2F0aW9uLlJlYWRXcml0ZS5Pd25lZEJ5IiwiVGVzdCJdfQ.zLwcvj4j07PG2IdiSXMJh-KL-uno9gndn0DY1GyzQbQ"

	oc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			ServicePrincipalProfile: api.ServicePrincipalProfile{
				TenantID:     "1234",
				ClientID:     "5678",
				ClientSecret: api.SecureString("shhh"),
			},
		},
	}

	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name    string
		mocks   func(token *mock_instancemetadata.MockServicePrincipalToken, tokenMaker *mockaad.MockTokenMaker)
		wantErr string
	}{
		{
			name: "has Application.ReadWrite.OwnedBy permission",
			mocks: func(token *mock_instancemetadata.MockServicePrincipalToken, tokenMaker *mockaad.MockTokenMaker) {
				tokenMaker.EXPECT().AuthenticateAndGetToken(ctx, log, oc, azure.PublicCloud.ResourceManagerEndpoint).Return(token, nil)
				token.EXPECT().OAuthToken().Return(invalidJWT)
			},
			wantErr: "400: InvalidServicePrincipalCredentials: properties.servicePrincipalProfile: The provided service principal must not have the Application.ReadWrite.OwnedBy permission.",
		},
		{
			name: "success",
			mocks: func(token *mock_instancemetadata.MockServicePrincipalToken, tokenMaker *mockaad.MockTokenMaker) {
				tokenMaker.EXPECT().AuthenticateAndGetToken(ctx, log, oc, azure.PublicCloud.ResourceManagerEndpoint).Return(token, nil)
				token.EXPECT().OAuthToken().Return(altsecidJWT)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			token := mock_instancemetadata.NewMockServicePrincipalToken(controller)
			tokenMaker := mockaad.NewMockTokenMaker(controller)
			tt.mocks(token, tokenMaker)

			dv := &openShiftClusterDynamicValidator{
				log:        log,
				tokenMaker: tokenMaker,
			}
			_, err := dv.validateServicePrincipalProfile(ctx, oc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestValidateProviders(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	dv := &openShiftClusterDynamicValidator{}

	for _, tt := range []struct {
		name    string
		mocks   func(*mockfeatures.MockProvidersClient)
		wantErr string
	}{
		{
			name: "all registered",
			mocks: func(providersClient *mockfeatures.MockProvidersClient) {
				providersClient.EXPECT().
					List(gomock.Any(), nil, "").
					Return([]mgmtfeatures.Provider{
						{
							Namespace:         to.StringPtr("Microsoft.Authorization"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Compute"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Network"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Storage"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("otherRegisteredProvider"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("otherNotRegisteredProvider"),
							RegistrationState: to.StringPtr("NotRegistered"),
						},
					}, nil)
			},
		},
		{
			name: "compute not registered",
			mocks: func(providersClient *mockfeatures.MockProvidersClient) {
				providersClient.EXPECT().
					List(gomock.Any(), nil, "").
					Return([]mgmtfeatures.Provider{
						{
							Namespace:         to.StringPtr("Microsoft.Authorization"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Compute"),
							RegistrationState: to.StringPtr("NotRegistered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Network"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Storage"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("otherRegisteredProvider"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("otherNotRegisteredProvider"),
							RegistrationState: to.StringPtr("NotRegistered"),
						},
					}, nil)
			},
			wantErr: "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Compute' is not registered.",
		},
		{
			name: "storage missing",
			mocks: func(providersClient *mockfeatures.MockProvidersClient) {
				providersClient.EXPECT().
					List(gomock.Any(), nil, "").
					Return([]mgmtfeatures.Provider{
						{
							Namespace:         to.StringPtr("Microsoft.Authorization"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Compute"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("Microsoft.Network"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("otherRegisteredProvider"),
							RegistrationState: to.StringPtr("Registered"),
						},
						{
							Namespace:         to.StringPtr("otherNotRegisteredProvider"),
							RegistrationState: to.StringPtr("NotRegistered"),
						},
					}, nil)
			},
			wantErr: "400: ResourceProviderNotRegistered: : The resource provider 'Microsoft.Storage' is not registered.",
		},
		{
			name: "error case",
			mocks: func(providersClient *mockfeatures.MockProvidersClient) {
				providersClient.EXPECT().
					List(gomock.Any(), nil, "").
					Return(nil, errors.New("random error"))
			},
			wantErr: "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			providerClient := mockfeatures.NewMockProvidersClient(controller)

			tt.mocks(providerClient)

			err := dv.validateProviders(ctx, providerClient)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestValidateVnet(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	resourceGroup := "testGroup"
	subnetName := "testSubnet"
	vnetName := "testVnet"
	validSubnet := fmt.Sprintf("/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnet/%s", resourceGroup, vnetName, subnetName)

	dv := &openShiftClusterDynamicValidator{}

	for _, tt := range []struct {
		name    string
		mocks   func(*mocknetwork.MockVirtualNetworksClient)
		oc      *api.OpenShiftCluster
		wantErr string
	}{
		{
			name: "fail: custom dns set",
			mocks: func(networksClient *mocknetwork.MockVirtualNetworksClient) {
				networksClient.EXPECT().
					Get(gomock.Any(), resourceGroup, vnetName, "").
					Return(mgmtnetwork.VirtualNetwork{
						VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
							DhcpOptions: &mgmtnetwork.DhcpOptions{
								DNSServers: &[]string{
									"172.16.1.1",
								},
							},
						},
					}, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided vnet '/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup/providers/Microsoft.Network/virtualNetworks/testVnet' is invalid: custom DNS servers are not supported.",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						SubnetID: validSubnet,
					},
				},
			},
		},
		{
			name: "pass: default settings",
			mocks: func(networksClient *mocknetwork.MockVirtualNetworksClient) {
				networksClient.EXPECT().
					Get(gomock.Any(), resourceGroup, vnetName, "").
					Return(mgmtnetwork.VirtualNetwork{
						VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
							DhcpOptions: &mgmtnetwork.DhcpOptions{
								DNSServers: &[]string{},
							},
						},
					}, nil)
			},
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						SubnetID: validSubnet,
					},
				},
			},
		},
		{
			name: "fail: vnet not found",
			mocks: func(networksClient *mocknetwork.MockVirtualNetworksClient) {
				networksClient.EXPECT().
					Get(gomock.Any(), resourceGroup, vnetName, "").
					Return(mgmtnetwork.VirtualNetwork{}, fmt.Errorf("The Resource 'Microsoft.Network/virtualNetworks/testVnet' under resource group 'testGroup' was not found."))
			},
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						SubnetID: validSubnet,
					},
				},
			},
			wantErr: "The Resource 'Microsoft.Network/virtualNetworks/testVnet' under resource group 'testGroup' was not found.",
		},
		{
			name: "fail: invalid subnet",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					MasterProfile: api.MasterProfile{
						SubnetID: "/invalid/subnet",
					},
				},
			},
			wantErr: `subnet ID "/invalid/subnet" has incorrect length`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			networkClient := mocknetwork.NewMockVirtualNetworksClient(controller)

			if tt.mocks != nil {
				tt.mocks(networkClient)
			}

			err := dv.validateVnet(ctx, networkClient, tt.oc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
