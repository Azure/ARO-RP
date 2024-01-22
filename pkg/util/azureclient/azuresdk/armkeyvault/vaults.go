package armkeyvault

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	sdkkeyvault "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
)

type VaultsClient interface {
	CheckNameAvailability(ctx context.Context, vaultName sdkkeyvault.VaultCheckNameAvailabilityParameters, options *sdkkeyvault.VaultsClientCheckNameAvailabilityOptions) (sdkkeyvault.VaultsClientCheckNameAvailabilityResponse, error)
}

type vaultsClient struct {
	*sdkkeyvault.VaultsClient
}

var _ VaultsClient = &vaultsClient{}

func NewVaultsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (VaultsClient, error) {
	client, err := sdkkeyvault.NewVaultsClient(subscriptionID, credential, options)
	return vaultsClient{
		VaultsClient: client,
	}, err
}
