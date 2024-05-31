package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"time"
)

var _ http.RoundTripper = (*AccessTokenTransport)(nil)

type AccessTokenTransport struct {
	AuthToken string
	transport http.RoundTripper
}

func NewAccessTokenTransport(authToken *AccessToken) *AccessTokenTransport {
	return &AccessTokenTransport{
		AuthToken: fmt.Sprintf("AccessToken %s", authToken),
		transport: &http.Transport{
			TLSHandshakeTimeout: time.Second * 5,
		},
	}
}

func (t *AccessTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.AuthToken)

	return t.transport.RoundTrip(req)
}
