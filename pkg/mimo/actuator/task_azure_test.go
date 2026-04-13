package actuator

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

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func TestAzureInit(t *testing.T) {
	require := require.New(t)

	controller := gomock.NewController(nil)
	_env := mock_env.NewMockInterface(controller)

	f := &th{env: _env}

	// no subscription document
	_, err := f.TokensClient()
	require.ErrorIs(err, errInvalidSubDoc)

	f.sub = &api.SubscriptionDocument{
		ID: "456",
		Subscription: &api.Subscription{Properties: &api.SubscriptionProperties{
			TenantID: "123",
		}},
	}

	// client cert credential creation failure
	_env.EXPECT().FPNewClientCertificateCredential(gomock.Eq("123"), gomock.Nil()).Return(nil, errors.New("oh no"))
	_, err = f.TokensClient()
	require.ErrorIs(err, errCreatingFpCredClusterTenant)

	// test successfully creating the client
	cred := &azcorefake.TokenCredential{}
	_env.EXPECT().FPNewClientCertificateCredential(gomock.Eq("123"), gomock.Nil()).Return(cred, nil)

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
	_env.EXPECT().ArmClientOptions(gomock.Any()).Return(
		&arm.ClientOptions{
			ClientOptions: azcore.ClientOptions{
				Transport: armcontainerregistryfake.NewTokensServerTransport(fakeTokens),
			},
		},
	)

	// Successfully create the client
	c, err := f.TokensClient()
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
}
