package graph

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"log"
	"net/http"
	"net/http/httputil"
	"os"

	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	kiotahttp "github.com/microsoft/kiota-http-go"
	core "github.com/microsoftgraph/msgraph-sdk-go-core"
)

var clientOptions = core.GraphClientOptions{
	GraphServiceVersion:        "", // v1 doesn't include the service version in the telemetry header
	GraphServiceLibraryVersion: "1.15.0",
}

// GraphRequestAdapter is the core service used by GraphBaseServiceClient to make requests to Microsoft Graph.
type GraphRequestAdapter struct {
	core.GraphRequestAdapterBase
}

const ENV_DEBUG_TRACE = "ARO_MSGRAPH_TRACE"

type DebugTransport struct {
	Transport http.RoundTripper
}

func (t *DebugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var data []byte

	data, _ = httputil.DumpRequestOut(req, true)
	log.Writer().Write(data)

	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	data, _ = httputil.DumpResponse(resp, true)
	log.Writer().Write(data)

	return resp, err
}

// NewGraphRequestAdapter creates a new GraphRequestAdapter with the given parameters
// Parameters:
// authenticationProvider: the provider used to authenticate requests
// Returns:
// a new GraphRequestAdapter
func NewGraphRequestAdapter(authenticationProvider absauth.AuthenticationProvider) (*GraphRequestAdapter, error) {
	//     The Graph service is not handling gzipped requests properly but Kiota's HTTP client gzips by default.
	//     This middleware list is equivalent to kiotahttp.GetDefaultMiddlewares, minus the CompressionHandler.
	middlewares := []kiotahttp.Middleware{
		kiotahttp.NewRetryHandler(),
		kiotahttp.NewRedirectHandler(),
		kiotahttp.NewParametersNameDecodingHandler(),
		kiotahttp.NewUserAgentHandler(),
	}

	httpClient := kiotahttp.GetDefaultClient(middlewares...)
	if _, doTrace := os.LookupEnv(ENV_DEBUG_TRACE); doTrace {
		httpClient.Transport = &DebugTransport{Transport: httpClient.Transport}
	}
	baseAdapter, err := core.NewGraphRequestAdapterBaseWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authenticationProvider, clientOptions, nil, nil, httpClient)
	if err != nil {
		return nil, err
	}
	result := &GraphRequestAdapter{
		GraphRequestAdapterBase: *baseAdapter,
	}

	return result, nil
}
