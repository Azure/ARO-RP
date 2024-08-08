package swagger

import "github.com/Azure/azure-sdk-for-go/sdk/azcore"

func NewSwaggerClient(azcoreClient *azcore.Client) *ManagedIdentityDataPlaneAPIClient {
	return &ManagedIdentityDataPlaneAPIClient{
		internal: azcoreClient,
	}
}
