package applens

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

const (
	userAgent = "ARO-AppLens-Client"
)

// ClientOptions defines the options for the AppLens client.
type ClientOptions struct {
	azcore.ClientOptions
}

func NewClientOptions() *ClientOptions {
	return &ClientOptions{
		azcore.ClientOptions{
			Retry: policy.RetryOptions{
				MaxRetries: 3,
				RetryDelay: time.Second * 10,
			},
			Telemetry: policy.TelemetryOptions{
				ApplicationID: userAgent,
				Disabled:      false,
			},
			Transport: &http.Client{
				Transport: &http.Transport{
					TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
					TLSClientConfig: &tls.Config{
						Renegotiation: tls.RenegotiateFreelyAsClient,
						MinVersion:    tls.VersionTLS12,
					},
				},
			},
		},
	}
}
