package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azcorefake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v2"
	armcontainerregistryfake "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v2/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	armnetworkfake "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6/fake"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const registryResourceID = "/subscriptions/93aeba23-2f76-4307-be82-02921df010cf/resourceGroups/global/providers/Microsoft.ContainerRegistry/registries/arointsvc"

func TestAzureInitRPTenant(t *testing.T) {
	require := require.New(t)
	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)

	f := &th{env: _env}

	// RP tenant client cert credential creation failure
	cred := &azcorefake.TokenCredential{}
	_env.EXPECT().TenantID().Return("789")
	_env.EXPECT().FPNewClientCertificateCredential(gomock.Eq("789"), gomock.Nil()).Return(nil, errors.New("oh no"))
	_, err := f.FirstPartyTokensClient()
	require.ErrorIs(err, errCreatingFpCredRPTenant)

	// add a fake in for testing
	fakeTokens := &armcontainerregistryfake.TokensServer{
		Get: func(ctx context.Context, resourceGroupName, registryName, tokenName string, options *armcontainerregistry.TokensClientGetOptions) (resp azcorefake.Responder[armcontainerregistry.TokensClientGetResponse], errResp azcorefake.ErrorResponder) {
			body := armcontainerregistry.TokensClientGetResponse{
				Token: armcontainerregistry.Token{
					Name: &tokenName,
					Properties: &armcontainerregistry.TokenProperties{
						Credentials: &armcontainerregistry.TokenCredentialsProperties{
							// Put the params in for some dirty testing :)
							Passwords: []*armcontainerregistry.TokenPassword{
								{
									Value: pointerutils.ToPtr(resourceGroupName),
								},
								{
									Value: pointerutils.ToPtr(registryName),
								},
								{
									Value: pointerutils.ToPtr(tokenName),
								},
							},
						},
					},
				},
			}
			resp.SetResponse(http.StatusOK, body, nil)
			return
		},
	}

	// Load the fake in via ArmClientOptions's ClientOptions.Transport
	_env.EXPECT().ArmClientOptions().Return(
		&arm.ClientOptions{
			ClientOptions: azcore.ClientOptions{
				Transport: armcontainerregistryfake.NewTokensServerTransport(fakeTokens),
			},
		},
	)

	// no azure clients client before we create it
	require.Nil(f.azRPTenant)

	// invalid ACR returns failure
	_env.EXPECT().TenantID().Return("789")
	_env.EXPECT().FPNewClientCertificateCredential(gomock.Eq("789"), gomock.Nil()).Return(cred, nil)
	_env.EXPECT().ACRResourceID().Return("")
	_, err = f.FirstPartyTokensClient()
	require.ErrorIs(err, errParsingACRResourceID)

	// we should have the azure clients now because that part succeeded
	require.NotNil(f.azRPTenant)

	// Successfully create the client
	_env.EXPECT().ACRResourceID().Return(registryResourceID)
	c, err := f.FirstPartyTokensClient()
	require.NoError(err)

	// Call the client w/ params and check that they're passed through
	p, err := c.GetTokenProperties(t.Context(), "a", "b", "c")
	require.NoError(err)
	require.Equal([]*armcontainerregistry.TokenPassword{
		{
			Value: pointerutils.ToPtr("a"),
		},
		{
			Value: pointerutils.ToPtr("b"),
		},
		{
			Value: pointerutils.ToPtr("c"),
		},
	}, p.Credentials.Passwords)

	// client created and cached
	require.NotNil(f.azRPTenant)
	require.NotNil(f.azRPTenant.tokensClient)
}

func TestAzureInitClusterTenant(t *testing.T) {
	require := require.New(t)
	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)

	f := &th{env: _env}

	// no subscription document
	_, err := f.LoadBalancersClient()
	require.ErrorIs(err, errInvalidSubDoc)

	f.sub = &api.SubscriptionDocument{
		ID: "456",
		Subscription: &api.Subscription{Properties: &api.SubscriptionProperties{
			TenantID: "123",
		}},
	}

	// cluster tenant client cert credential creation failure
	_env.EXPECT().FPNewClientCertificateCredential(gomock.Eq("123"), gomock.Nil()).Return(nil, errors.New("oh no"))
	_, err = f.LoadBalancersClient()
	require.ErrorIs(err, errCreatingFpCredClusterTenant)

	// add a fake in for testing
	fakeLB := &armnetworkfake.LoadBalancersServer{
		Get: func(ctx context.Context, resourceGroupName, loadBalancerName string, options *armnetwork.LoadBalancersClientGetOptions) (resp azcorefake.Responder[armnetwork.LoadBalancersClientGetResponse], errResp azcorefake.ErrorResponder) {
			body := armnetwork.LoadBalancersClientGetResponse{
				LoadBalancer: armnetwork.LoadBalancer{
					Name: pointerutils.ToPtr(loadBalancerName),
					ID: pointerutils.ToPtr(azure.Resource{
						SubscriptionID: "456",
						ResourceGroup:  resourceGroupName,
						Provider:       "Microsoft.Network",
						ResourceType:   "loadBalancers",
						ResourceName:   loadBalancerName,
					}.String()),
				},
			}
			resp.SetResponse(http.StatusOK, body, nil)
			return
		},
	}

	// Load the fake in via ArmClientOptions's ClientOptions.Transport
	_env.EXPECT().ArmClientOptions().Return(
		&arm.ClientOptions{
			ClientOptions: azcore.ClientOptions{
				Transport: armnetworkfake.NewLoadBalancersServerTransport(fakeLB),
			},
		},
	)

	// no azure clients client before we create it
	require.Nil(f.azClusterTenant)

	// Successfully create the client
	cred := &azcorefake.TokenCredential{}
	_env.EXPECT().FPNewClientCertificateCredential(gomock.Eq("123"), gomock.Nil()).Return(cred, nil)
	c, err := f.LoadBalancersClient()
	require.NoError(err)

	// Call the client w/ params and check that they're passed through
	p, err := c.Get(t.Context(), "a", "b", nil)
	require.NoError(err)
	require.Equal(
		armnetwork.LoadBalancer{
			Name: pointerutils.ToPtr("b"),
			ID:   pointerutils.ToPtr("/subscriptions/456/resourceGroups/a/providers/Microsoft.Network/loadBalancers/b"),
		},
		p.LoadBalancer)

	// client created and cached
	require.NotNil(f.azClusterTenant)
	require.NotNil(f.azClusterTenant.loadBalancerClient)
}
