package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

func adminRequest(ctx context.Context, method, path string, params url.Values, in, out interface{}) (*http.Response, error) {
	if os.Getenv("RP_MODE") != "development" {
		return nil, errors.New("only development RP mode is supported")
	}

	adminAPIBaseURI := "https://localhost:8443/admin"
	adminURL, err := url.Parse(adminAPIBaseURI + path)
	if err != nil {
		return nil, err
	}
	adminURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, method, adminURL.String(), nil)
	if err != nil {
		return nil, err
	}

	if in != nil {
		buf := &bytes.Buffer{}
		err := json.NewEncoder(buf).Encode(in)
		if err != nil {
			return nil, err
		}
		req.Body = ioutil.NopCloser(buf)
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		resp.Body.Read(nil)
		resp.Body.Close()
	}()

	if out != nil && resp.Header.Get("Content-Type") == "application/json" {
		return resp, json.NewDecoder(resp.Body).Decode(&out)
	}

	return resp, nil
}
