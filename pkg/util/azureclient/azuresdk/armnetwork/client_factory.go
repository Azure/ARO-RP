package armnetwork

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func NewClientFactory(env *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (*armnetwork.ClientFactory, error) {
	options := arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: env.Cloud,
		},
	}
	return armnetwork.NewClientFactory(subscriptionID, credential, &options)
}
