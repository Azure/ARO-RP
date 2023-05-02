package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/Azure/go-autorest/autorest"
)

func DecorateSenderWithLogging(sender autorest.Sender) autorest.Sender {
	loggingDecorator := LoggingDecorator()
	return autorest.DecorateSender(sender, loggingDecorator, autorest.DoCloseIfError())
}

// LoggingDecorator returns a function which is used to wrap and modify the behaviour of an autorest.Sender.
// Azure Clients will have the sender wrapped by that function
// in order to intercept http calls using our custom RoundTripper (through the adapter).
func LoggingDecorator() autorest.SendDecorator {
	return func(s autorest.Sender) autorest.Sender {
		rt := NewCustomRoundTripper(
			&roundTripperAdapter{Sender: s},
		)
		return autorest.SenderFunc(rt.RoundTrip)
	}
}

// roundTripperAdapter converts from autorest.Sender (internal field)
// to http.RoundTripper (external method).
type roundTripperAdapter struct {
	Sender autorest.Sender
}

func (rta *roundTripperAdapter) RoundTrip(req *http.Request) (*http.Response, error) {
	return rta.Sender.Do(req)
}
