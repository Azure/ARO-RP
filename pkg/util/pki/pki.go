package pki

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
)

var caMap map[string]x509.CertPool = make(map[string]x509.CertPool)
var mu sync.RWMutex

type RootCAs struct {
	RootsInfos []RootInfo `json:"RootsInfos"`
}

type RootInfo struct {
	RootName      string             `json:"rootName"`
	CaName        string             `json:"CaName"`
	Cdp           string             `json:"Cdp"`
	AppType       string             `json:"AppType"`
	StoreLocation string             `json:"StoreLocation"`
	StoreName     string             `json:"StoreName"`
	Body          string             `json:"Body"`
	PEM           string             `json:"PEM"`
	Intermediates []IntermediateInfo `json:"Intermediates"`
}

type IntermediateInfo struct {
	IntermediateName string `json:"IntermediateName"`
	AppType          string `json:"AppType"`
	Cdp              string `json:"Cdp"`
	StoreLocation    string `json:"StoreLocation"`
	StoreName        string `json:"StoreName"`
	Body             string `json:"Body"`
	PEM              string `json:"PEM"`
}

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

	if err != nil {
		return nil, err
	}

	// Read in certs from endpoint
	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	var rootCAs RootCAs
	json.Unmarshal(body, &rootCAs)
	return &rootCAs, nil
}

func GetTlsCertPool(urlTemplate, caName string) (*x509.CertPool, error) {
	url := fmt.Sprintf(urlTemplate, caName)
	caCertPool, ok := getCaCertPoolFromMap(url)
	if ok {
		return &caCertPool, nil
	} else {
		caCertPool, err := buildCertPoolForCaName(url)

		if err != nil || caCertPool == nil {
			return nil, err
		}

		setCaCertPoolInMap(url, *caCertPool)

		return caCertPool, nil
	}
}

func getCaCertPoolFromMap(key string) (x509.CertPool, bool) {
	mu.RLock()
	defer mu.RUnlock()
	caCertPool, ok := caMap[key]
	return caCertPool, ok
}

func setCaCertPoolInMap(key string, caCertPool x509.CertPool) {
	mu.Lock()
	defer mu.Unlock()
	caMap[key] = caCertPool
}

func buildCertPoolForCaName(url string) (*x509.CertPool, error) {
	data, err := FetchDataFromGetIssuerPki(url)

	if err != nil {
		return nil, err
	}

	// Create a CertPool
	caCertPool, err := x509.SystemCertPool()

	if err != nil {
		return nil, err
	}

	for _, rootInfo := range data.RootsInfos {
		caCert := rootInfo.PEM

		// Append the custom CA certificate to the CertPool
		if ok := caCertPool.AppendCertsFromPEM([]byte(caCert)); !ok {
			return nil, errors.New("failed to load cert into cert pool")
		}

		for _, intermediate := range rootInfo.Intermediates {
			intermediateCert := intermediate.PEM

			// Append the custom CA certificate to the CertPool
			if ok := caCertPool.AppendCertsFromPEM([]byte(intermediateCert)); !ok {
				return nil, errors.New("failed to load cert into cert pool")
			}
		}
	}

	return caCertPool, nil
}
