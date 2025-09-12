package pki

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"io"
	"net/http"
)

// https://aka.ms/getissuers
// The v3 endpoint can be used to get ca certs
// For example https://issuer.pki.azure.com/dsms/issuercertificates?getissuersv3&caName=ame
// returns the ame certs
func FetchDataFromGetIssuerPki(url string) (*RootCAs, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var rootCAs RootCAs
	json.Unmarshal(body, &rootCAs)
	return &rootCAs, nil
}

// CombineRootCAs is a helper function to use to combine multiple RootCAs instances
// into one.
func CombineRootCAs(rootCAs ...*RootCAs) *RootCAs {
	out := &RootCAs{}

	for _, rc := range rootCAs {
		out.RootsInfos = append(out.RootsInfos, rc.RootsInfos...)
	}

	return out
}

// FetchDataFromGetIssuersPkiUrls is a helper function that leverages FetchDataFromGetIssuerPki
// and CombineRootCAs to return a single RootCAs instance containing the certs retrieved from
// multiple issuer PKI URLs.
func FetchDataFromGetIssuerPkiUrls(urls []string) (*RootCAs, error) {
	var rootCAs []*RootCAs

	for _, url := range urls {
		rc, err := FetchDataFromGetIssuerPki(url)
		if err != nil {
			return nil, err
		}

		rootCAs = append(rootCAs, rc)
	}

	return CombineRootCAs(rootCAs...), nil
}
