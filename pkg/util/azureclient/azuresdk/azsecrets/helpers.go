package azsecrets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"

	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
)

func URI(metadata instancemetadata.InstanceMetadata, suffix, keyVaultPrefix string) string {
	return fmt.Sprintf("https://%s%s.%s/", keyVaultPrefix, suffix, metadata.Environment().KeyVaultDNSSuffix)
}

// ParseSecretAsCertificate parses the value of a KeyVault secret as a set of PEM blocks containing a private key and certificate.
func ParseSecretAsCertificate(secret azsecrets.GetSecretResponse) (*rsa.PrivateKey, []*x509.Certificate, error) {
	if secret.Value == nil {
		return nil, nil, errors.New("secret response has no value")
	}

	key, certs, err := utilpem.Parse([]byte(*secret.Value))
	if err != nil {
		return nil, nil, err
	}

	if key == nil {
		return nil, nil, fmt.Errorf("no private key found")
	}

	if len(certs) == 0 {
		return nil, nil, fmt.Errorf("no certificate found")
	}

	return key, certs, nil
}

// ExtractBase64Value extracts the value of a KeyVault secret as a base64-encoded string
func ExtractBase64Value(secret azsecrets.GetSecretResponse) ([]byte, error) {
	if secret.Value == nil {
		return nil, errors.New("secret response has no value")
	}
	return base64.StdEncoding.DecodeString(*secret.Value)
}
