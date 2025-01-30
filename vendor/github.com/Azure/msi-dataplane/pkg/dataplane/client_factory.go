package dataplane

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/msi-dataplane/pkg/dataplane/internal"
)

const (
	// TODO - Make module name configurable
	moduleName = "managedidentitydataplane.APIClient"
	// TODO - Tie the module version to update automatically with new releases
	moduleVersion = "v0.0.1"
)

// ClientFactory creates clients for managed identity credentials.
type ClientFactory interface {
	// NewClient creates a client that can operate on credentials for one managed identity.
	// identityURL is the x-ms-identity-url header provided from ARM, including any path,
	// query parameters, etc.
	NewClient(identityURL string) (Client, error)
}

// NewClientFactory creates a new MSI data plane client factory. The credentials and audience presented
// are for the first-party credential. As the server to be contacted for each identity varies, a factory
// is returned that can create clients on-demand.
func NewClientFactory(cred azcore.TokenCredential, audience string, opts *azcore.ClientOptions) (ClientFactory, error) {
	azCoreClient, err := azcore.NewClient(moduleName, moduleVersion, runtime.PipelineOptions{
		PerCall: []policy.Policy{newAuthenticatorPolicy(cred, audience)},
	}, opts)
	if err != nil {
		return nil, fmt.Errorf("error creating azcore client: %w", err)
	}
	return &clientFactory{delegate: azCoreClient}, nil
}

type clientFactory struct {
	delegate *azcore.Client
}

var _ ClientFactory = (*clientFactory)(nil)

type httpRequestDoerFunc func(*http.Request) (*http.Response, error)

var _ internal.HttpRequestDoer = (httpRequestDoerFunc)(nil)

func (f httpRequestDoerFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func (c *clientFactory) NewClient(identityURL string) (Client, error) {
	parsedURL, err := url.ParseRequestURI(identityURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing identity URL: %w", err)
	}
	server := url.URL{
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
		Path:   parsedURL.Path,
	}

	client, err := internal.NewClientWithResponses(
		server.String(),
		internal.WithHTTPClient(httpRequestDoerFunc(func(req *http.Request) (*http.Response, error) {
			azreq, err := runtime.NewRequestFromRequest(req)
			if err != nil {
				return nil, err
			}
			return c.delegate.Pipeline().Do(azreq)
		})),
		internal.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			// x-ms-identity-url header from ARM contains query parameters we need to keep
			query := req.URL.Query()
			for key, values := range parsedURL.Query() {
				for _, value := range values {
					query.Add(key, value)
				}
			}
			req.URL.RawQuery = query.Encode()

			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}
	return &clientAdapter{delegate: client}, nil
}
