package dbtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/env"
)

type Client interface {
	Token(context.Context, string) (string, error)
}

type doer interface {
	Do(*http.Request) (*http.Response, error)
}

type client struct {
	c          doer
	authorizer autorest.Authorizer
	url        string
}

func NewClient(_env env.Core, authorizer autorest.Authorizer, insecureSkipVerify bool) (Client, error) {
	url := "https://localhost:8445"
	if !_env.IsLocalDevelopmentMode() {
		if err := env.ValidateVars("DBTOKEN_URL"); err != nil {
			return nil, err
		}

		url = os.Getenv("DBTOKEN_URL")
	}

	return &client{
		c: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: insecureSkipVerify,
				},
				// disable HTTP/2 for now: https://github.com/golang/go/issues/36026
				TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{},
			},
		},
		authorizer: authorizer,
		url:        url,
	}, nil
}

func (c *client) Token(ctx context.Context, permission string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/token", nil)
	if err != nil {
		return "", err
	}

	q := url.Values{
		"permission": []string{permission},
	}
	req.URL.RawQuery = q.Encode()

	var tr *tokenResponse
	err = c.do(req, &tr)
	if err != nil {
		return "", err
	}

	return tr.Token, nil
}

func (c *client) do(req *http.Request, i interface{}) (err error) {
	req, err = autorest.Prepare(req, c.authorizer.WithAuthorization())
	if err != nil {
		return err
	}

	resp, err := c.c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		return fmt.Errorf("unexpected content type %q", resp.Header.Get("Content-Type"))
	}

	return json.NewDecoder(resp.Body).Decode(&i)
}
