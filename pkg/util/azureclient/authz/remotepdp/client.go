package remotepdp

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

// RemotePDPClient represents the Microsoft Remote PDP API Spec
type RemotePDPClient interface {
	CheckAccess(context.Context, AuthorizationRequest) (*AuthorizationDecisionResponse, error)
}

// remotePDPClient implements RemotePDPClient
type remotePDPClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

// NewRemotePDPClient returns an implementation of RemotePDPClient
// endpoint - the fqdn of the regional specific endpoint of PDP
// scope - the oauth scope required by the PDP serer
// cred - the credential of the client to call the PDP server
func NewRemotePDPClient(endpoint, scope string, cred azcore.TokenCredential) RemotePDPClient {
	authPolicy := runtime.NewBearerTokenPolicy(cred, []string{scope}, nil)

	pipeline := runtime.NewPipeline(
		modulename,
		version,
		runtime.PipelineOptions{
			PerCall:  []policy.Policy{},
			PerRetry: []policy.Policy{authPolicy},
		},
		nil,
	)

	return &remotePDPClient{endpoint, pipeline}
}

// CheckAccess sends an Authorization query to the PDP server specified in the client
// ctx - the context to propagate
// authzReq - the actual AuthorizationRequest
func (r *remotePDPClient) CheckAccess(ctx context.Context, authzReq AuthorizationRequest) (*AuthorizationDecisionResponse, error) {
	req, err := runtime.NewRequest(ctx, http.MethodPost, r.endpoint)
	if err != nil {
		return nil, err
	}
	runtime.MarshalAsJSON(req, authzReq)

	res, err := r.pipeline.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, newCheckAccessError(res)
	}

	var accessDecision AuthorizationDecisionResponse
	err = runtime.UnmarshalAsJSON(res, &accessDecision)

	if err != nil {
		return nil, err
	}
	return &accessDecision, nil
}

// newCheckAccessError returns an error when non HTTP 200 response is returned.
func newCheckAccessError(r *http.Response) error {
	resErr := azcore.ResponseError{
		StatusCode:  r.StatusCode,
		RawResponse: r,
	}
	payload, err := runtime.Payload(r)
	if err != nil {
		return err
	}
	var checkAccessError CheckAccessErrorResponse
	err = json.Unmarshal(payload, &checkAccessError)
	if err != nil {
		return err
	}
	resErr.ErrorCode = checkAccessError.Message
	return &resErr
}
