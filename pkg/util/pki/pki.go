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

// https://aka.ms/getissuers
// The v3 endpoint can be used to get ca certs
// For example https://issuer.pki.azure.com/dsms/issuercertificates?getissuersv3&caName=ame
// returns the ame certs
func buildCertPoolForCaName(url string) (*x509.CertPool, error) {
	response, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	// Create a CertPool
	caCertPool, err := x509.SystemCertPool()

	if err != nil {
		return nil, err
	}

	// Read in certs from endpoint
	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	json.Unmarshal(body, &data)

	roots := data["RootsInfos"].([]interface{})
	for _, root := range roots {
		caCert := root.(map[string]interface{})["PEM"].(string)

		// Append the custom CA certificate to the CertPool
		if ok := caCertPool.AppendCertsFromPEM([]byte(caCert)); !ok {
			return nil, errors.New("failed to load cert into cert pool")
		}

		intermediates := root.(map[string]interface{})["Intermediates"].([]interface{})
		for _, intermediate := range intermediates {
			intermediateCert := intermediate.(map[string]interface{})["PEM"].(string)

			// Append the custom CA certificate to the CertPool
			if ok := caCertPool.AppendCertsFromPEM([]byte(intermediateCert)); !ok {
				return nil, errors.New("failed to load cert into cert pool")
			}
		}
	}

	return caCertPool, nil
}
