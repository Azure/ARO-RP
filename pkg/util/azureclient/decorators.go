package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/common"
)

// DecorateSenderWithLogging decorates a sender in order to intercept HTTP calls using a custom RoundTripper
// and log low level HTTP request's information.
func DecorateSenderWithLogging(sender autorest.Sender) autorest.Sender {
	decorators := []autorest.SendDecorator{loggingDecorator(), autorest.DoCloseIfError()}
	if d := common.NewFirstFailSendDecorator(); d != nil {
		// fault injector is appended last, making it outermost in the autorest chain (runs first).
		// The injector returns a synthetic HTTP response and logs each injection directly.
		decorators = append(decorators, d)
	}
	return autorest.DecorateSender(sender, decorators...)
}

// loggingDecorator returns a function which is used to wrap and modify the behaviour of an autorest.Sender.
// Azure Clients will have the sender wrapped by that function
// in order to intercept http calls using our custom RoundTripper (through the adapter).
func loggingDecorator() autorest.SendDecorator {
	if outboundHTTPLoggingEnabled() {
		return func(s autorest.Sender) autorest.Sender {
			return autorest.SenderFunc(func(req *http.Request) (*http.Response, error) {
				return loggingRoundTripper(req, func() (*http.Response, error) {
					return s.Do(req)
				})
			})
		}
	}
	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(req *http.Request) (*http.Response, error) {
			return errorOnlyLoggingRoundTripper(req, func() (*http.Response, error) {
				return s.Do(req)
			})
		})
	}
}
