package remotepdp

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

type RemotePDPClient interface {
	CheckAccess(context.Context, AuthorizationRequest) (*AuthorizationDecisionResponse, error)
}

// TODO insert the required attributes
type remotePDPClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

// TODO Insert the required parameters
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

// TODO Implement it
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
