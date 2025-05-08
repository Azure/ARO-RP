package pki

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"errors"
)

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

func BuildCertPoolForCaName(data *RootCAs) (*x509.CertPool, error) {
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
