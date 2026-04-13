package actuator

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v2"
	armcontainerregistryfake "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v2/fake"
)

func TestAzureInit(t *testing.T) {
	require := require.New(t)

	controller := gomock.NewController(nil)
	_env := mock_env.NewMockInterface(controller)
    azenv := AROEnvironment

	f := &th{env: _env}

	// no subscription document
	_, err := f.TokensClient()
	require.ErrorIs(err, errInvalidSubDoc)

	f.sub = &api.SubscriptionDocument{Subscription: &api.Subscription{Properties: &api.SubscriptionProperties{
		TenantID: "123",
	}}}

	// client cert credential creation failure
	_env.EXPECT().FPNewClientCertificateCredential(gomock.Eq("123"), gomock.Nil()).Return(nil, errors.New("oh no"))
	_, err = f.TokensClient()
	require.ErrorIs(err, errCreatingFpCredClusterTenant)

	// successfully created
	_env.EXPECT().FPNewClientCertificateCredential(gomock.Eq("123"), gomock.Nil()).Return(&azidentity.ClientCertificateCredential{}, nil)

	// add a fake in for testing
	fakeTokens := armcontainerregistryfake.TokensServer{
		Get: func(ctx context.Context, resourceGroupName, registryName, tokenName string, options *armcontainerregistry.TokensClientGetOptions) (resp fake.Responder[armcontainerregistry.TokensClientGetResponse], errResp fake.ErrorResponder) {
			body := armcontainerregistry.TokensClientGetResponse{
				Token: armcontainerregistry.Token{
					Properties: &armcontainerregistry.TokenProperties{},
				},
			}
			resp.SetResponse(http.StatusOK, body, nil)
			return
		},
	}

    _env.EXPECT().Environment().Return()

	c, err := f.TokensClient()
	require.NoError(err)

	c.GetTokenProperties()

}
