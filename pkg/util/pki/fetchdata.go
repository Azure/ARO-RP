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
