package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/Azure/go-autorest/autorest"
)

// DecorateSenderWithLogging decorates a sender in order to intercept HTTP calls using a custom RoundTripper
// and log low level HTTP request's information.
func DecorateSenderWithLogging(sender autorest.Sender) autorest.Sender {
	loggingDecorator := loggingDecorator()
	return autorest.DecorateSender(sender, loggingDecorator, autorest.DoCloseIfError())
}

// loggingDecorator returns a function which is used to wrap and modify the behaviour of an autorest.Sender.
// Azure Clients will have the sender wrapped by that function
// in order to intercept http calls using our custom RoundTripper (through the adapter).
func loggingDecorator() autorest.SendDecorator {
	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(req *http.Request) (*http.Response, error) {
			return loggingRoundTripper(req, func() (*http.Response, error) {
				return s.Do(req)
			})
		})
	}
}
