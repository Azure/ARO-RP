package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
)

type RequestBuilder struct {
	method   string
	baseURL  string
	endpoint string
	headers  map[string]string
	params   url.Values
	body     []byte
	ctx      context.Context
}

func NewRequestBuilder(method, baseURL string) *RequestBuilder {
	return &RequestBuilder{
		method:  method,
		baseURL: baseURL,
		headers: make(map[string]string),
		params:  url.Values{},
		ctx:     context.Background(),
	}
}

func (rb *RequestBuilder) SetEndpoint(endpoint string) *RequestBuilder {
	rb.endpoint = endpoint
	return rb
}

func (rb *RequestBuilder) AddHeader(key, value string) *RequestBuilder {
	rb.headers[key] = value
	return rb
}

func (rb *RequestBuilder) AddParam(key, value string) *RequestBuilder {
	rb.params.Add(key, value)
	return rb
}

func (rb *RequestBuilder) SetBody(body []byte) *RequestBuilder {
	rb.body = body
	return rb
}

func (rb *RequestBuilder) SetContext(ctx context.Context) *RequestBuilder {
	rb.ctx = ctx
	return rb
}

func (rb *RequestBuilder) Build() (*http.Request, error) {
	parsedURL, err := url.Parse(rb.baseURL)
	if err != nil {
		return nil, err
	}
	parsedURL.Path = rb.endpoint
	parsedURL.RawQuery = rb.params.Encode()

	req, err := http.NewRequestWithContext(rb.ctx, rb.method, parsedURL.String(), bytes.NewBuffer(rb.body))
	if err != nil {
		return nil, err
	}

	for key, value := range rb.headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
