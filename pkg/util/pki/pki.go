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
)

var caMap map[string]x509.CertPool

type Pki interface {
	GetTlsCertPool(caName string) (*x509.CertPool, error)
}

type pki struct {
	urlTemplate string
}

func NewPki(kpiUrl string) Pki {
	if caMap == nil {
		caMap = make(map[string]x509.CertPool)
	}

	return &pki{
		urlTemplate: kpiUrl,
	}
}

func (k *pki) GetTlsCertPool(caName string) (*x509.CertPool, error) {
	if caCertPool, ok := caMap[caName]; ok {
		return &caCertPool, nil
	} else {
		caCertPool, err := buildCertPoolForCaName(k.urlTemplate, caName)

		if err != nil || caCertPool == nil {
			return nil, err
		}

		caMap[caName] = *caCertPool

		return caCertPool, nil
	}
}

// https://aka.ms/getissuers
// The v3 endpoint can be used to get ca certs
// For example https://issuer.pki.azure.com/dsms/issuercertificates?getissuersv3&caName=ame
// returns the ame certs
func buildCertPoolForCaName(baseUrl, caName string) (*x509.CertPool, error) {
	url := fmt.Sprintf(baseUrl, caName)
	response, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	// Create a new x509.CertPool
	caCertPool := x509.NewCertPool()

	// Read in certs from endpoint
	body, _ := ioutil.ReadAll(response.Body)
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
