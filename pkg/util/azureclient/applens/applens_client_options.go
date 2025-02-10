package applens

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

const (
	userAgent = "ARO-AppLens-Client"
)

// ClientOptions defines the options for the AppLens client.
type ClientOptions struct {
	azcore.ClientOptions
}

func NewClientOptions(certPool *x509.CertPool) *ClientOptions {
	httpTransport := &http.Transport{
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		TLSClientConfig: &tls.Config{
			RootCAs: certPool,
		},
	}

	httpClient := &http.Client{
		Transport: httpTransport,
	}

	return &ClientOptions{
		azcore.ClientOptions{
			Retry: policy.RetryOptions{
				MaxRetries: 3,
				// ARO-2567
				// If the retry logic takes longer than 60 seconds,
				// the correct error message will not be captured.
				// With a setting of 3 seconds it was erroring out
				// in ~30 seconds (3 seconds + 12 seconds + round
				// trip / response time from all 3 calls).
				RetryDelay: time.Second * 3,
			},
			Telemetry: policy.TelemetryOptions{
				ApplicationID: userAgent,
				Disabled:      false,
			},
			PerCallPolicies: []policy.Policy{azureclient.NewLoggingPolicy()},
			Transport:       httpClient,
		},
	}
}
