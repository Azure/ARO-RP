package kubeadminkubeconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func parseTokenResponse(location string) (string, error) {
	locURL, err := url.Parse(location)
	if err != nil {
		return "", err
	}

	// wow, sometimes openshift-oauth returns us errors via the query and not
	// the fragment.  I don't know if that's spec-compliant but we check it...
	v, err := url.ParseQuery(locURL.RawQuery)
	if err != nil {
		return "", err
	}

	if v.Get("error") != "" {
		return "", fmt.Errorf("%s", v.Get("error_description"))
	}

	v, err = url.ParseQuery(locURL.Fragment)
	if err != nil {
		return "", err
	}

	if v.Get("error") != "" {
		return "", fmt.Errorf("%s", v.Get("error_description"))
	}

	return v.Get("access_token"), nil
}

func getTokenURLFromConsoleURL(consoleURL string) (*url.URL, error) {
	tokenURL, err := url.Parse(consoleURL)
	if err != nil {
		return nil, err
	}

	tokenURL.Host = strings.Replace(tokenURL.Host, "console-openshift-console", "oauth-openshift", 1)
	tokenURL.Path = "/oauth/authorize"

	q := tokenURL.Query()
	q.Set("response_type", "token")
	q.Set("client_id", "openshift-challenging-client")
	tokenURL.RawQuery = q.Encode()

	return tokenURL, nil
}

func getAuthorizedToken(ctx context.Context, tokenURL *url.URL, username, password string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL.String(), nil)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(username, password)
	req.Header.Add("X-CSRF-Token", "1")

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}

	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return parseTokenResponse(resp.Header.Get("Location"))
}
