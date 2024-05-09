package armnetwork

import (
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func NewClientFactory(env *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (*armnetwork.ClientFactory, error) {
	options := arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: env.Cloud,
			Retry: policy.RetryOptions{
				ShouldRetry: isRetriable,
			},
		},
	}
	return armnetwork.NewClientFactory(subscriptionID, credential, &options)
}

// isRetriable checks if the response is retriable.
func isRetriable(resp *http.Response, err error) bool {
	if err != nil {
		return false
	}
	// Don't retry if successful
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return false
	}
	// Retry if the status code is retriable
	for _, sc := range autorest.StatusCodesForRetry {
		if resp.StatusCode == sc {
			return true
		}
	}

	// Check if the body contains the certain strings that can be retried.
	var b []byte
	_, err = resp.Body.Read(b)
	if err != nil {
		return true
	}
	body := string(b)
	return strings.Contains(body, "AADSTS7000215") ||
		strings.Contains(body, "AADSTS7000216") ||
		strings.Contains(body, "AuthorizationFailed")
}
