package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func parseTokenResponse(location string) (string, error) {
	locURL, err := url.Parse(location)
	if err != nil {
		return "", err
	}

	for _, param := range strings.Split(locURL.Fragment, "&") {
		nameValue := strings.Split(param, "=")
		if nameValue[0] == "access_token" {
			return nameValue[1], nil
		}
	}
	return "", fmt.Errorf("token not found in response")
}

func getTokenURLFromConsoleURL(consoleURL string) (*url.URL, error) {
	tokenURL, err := url.Parse(strings.Replace(consoleURL, "console-openshift-console.apps.", "oauth-openshift.apps.", 1))
	if err != nil {
		return nil, err
	}
	tokenURL.Path = "/oauth/authorize"
	q := tokenURL.Query()
	q.Set("response_type", "token")
	q.Set("client_id", "openshift-challenging-client")
	tokenURL.RawQuery = q.Encode()
	return tokenURL, nil
}

func getAuthorizedToken(tokenURL *url.URL, username, password string) (string, error) {
	req, err := http.NewRequest("GET", tokenURL.String(), nil)
	req.SetBasicAuth(username, password)
	req.Header.Add("X-CSRF-Token", "1")

	resp, err := (&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 302 {
		return "", err
	}

	loc := resp.Header["Location"]
	if loc == nil {
		return "", fmt.Errorf("no Location header found")
	}
	return parseTokenResponse(loc[0])
}
