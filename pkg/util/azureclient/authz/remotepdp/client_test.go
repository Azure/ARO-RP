package remotepdp

import (
	"context"
	"net/http"
	"testing"

	testhttp "github.com/Azure/ARO-RP/test/util/http"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

func TestSuccessfulCallReturnsADecision(t *testing.T) {
	srv, close := testhttp.NewTLSServer()
	defer close()
	srv.SetResponse(
		testhttp.WithStatusCode(http.StatusOK),
	)

	client := createClientWithServer(srv)

	decision, err := client.CheckAccess(context.Background(), AuthorizationRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if decision == nil {
		t.Error("Successful calls should return an access decision")
	}
}

func TestFailedCallReturns(t *testing.T) {
	srv, close := testhttp.NewTLSServer()
	defer close()
	srv.SetResponse(
		testhttp.WithStatusCode(http.StatusUnauthorized),
	)

	client := createClientWithServer(srv)

	_, err := client.CheckAccess(context.Background(), AuthorizationRequest{})
	if err == nil {
		t.Error("Call resulting in a failure should return an error")
	}

}

func createClientWithServer(s *testhttp.Server) RemotePDPClient {
	return &remotePDPClient{
		endpoint: s.URL(),
		pipeline: runtime.NewPipeline(
			"remotepdpclient_test",
			"v1.0.0",
			runtime.PipelineOptions{},
			&policy.ClientOptions{Transport: s},
		),
	}

}
