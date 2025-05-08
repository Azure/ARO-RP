package oidcbuilder

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/pkg/errors"

	"github.com/Azure/ARO-RP/pkg/env"
)

func CreateKeyPair(env env.Interface) (encPrivateKey []byte, encPublicKey []byte, err error) {
	// Generate RSA keypair
	privateKey, err := rsa.GenerateKey(rand.Reader, env.OIDCKeyBitSize())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate private key")
	}
	encodedPrivateKey := pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Serialize public key into a byte array to prepare to store it in the OIDC storage blob
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to serialize public key")
	}
	encodedPublicKey := pem.EncodeToMemory(&pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   pubKeyBytes,
	})

	return encodedPrivateKey, encodedPublicKey, nil
}

type JSONWebKeySet struct {
	Keys []jose.JSONWebKey `json:"keys"`
}

// buildJSONWebKeySet builds JSON web key set from the public key
func BuildJSONWebKeySet(publicKeyContent []byte) ([]byte, error) {
	block, _ := pem.Decode(publicKeyContent)
	if block == nil {
		return nil, errors.Errorf("Failed to decode PEM file")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to parse key content")
	}

	var alg jose.SignatureAlgorithm
	switch publicKey.(type) {
	case *rsa.PublicKey:
		alg = jose.RS256
	default:
		return nil, errors.Errorf("Public key is not of type RSA")
	}

	kid, err := keyIDFromPublicKey(publicKey)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch key ID from public key")
	}

	var keys []jose.JSONWebKey
	keys = append(keys, jose.JSONWebKey{
		Key:       publicKey,
		KeyID:     kid,
		Algorithm: string(alg),
		Use:       "sig",
	})

	keySet, err := json.MarshalIndent(JSONWebKeySet{Keys: keys}, "", "    ")
	if err != nil {
		return nil, errors.Wrapf(err, "JSON encoding of web key set failed")
	}

	return keySet, nil
}

// keyIDFromPublicKey derives a key ID non-reversibly from a public key
// reference: https://github.com/kubernetes/kubernetes/blob/v1.21.0/pkg/serviceaccount/jwt.go#L89-L111
func keyIDFromPublicKey(publicKey interface{}) (string, error) {
	publicKeyDERBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to serialize public key to DER format")
	}

	hasher := crypto.SHA256.New()
	hasher.Write(publicKeyDERBytes)
	publicKeyDERHash := hasher.Sum(nil)

	keyID := base64.RawURLEncoding.EncodeToString(publicKeyDERHash)

	return keyID, nil
}

const (
	discoveryDocumentTemplate = `{
		"issuer": "%s",
		"jwks_uri": "%s/openid/v1/jwks",
		"response_types_supported": [
			"id_token"
		],
		"subject_types_supported": [
			"public"
		],
		"id_token_signing_alg_values_supported": [
			"RS256"
		],
		"claims_supported": [
			"aud",
			"exp",
			"sub",
			"iat",
			"iss",
			"sub"
		]
	}`
)

func GenerateDiscoveryDocument(bucketURL string) string {
	return fmt.Sprintf(discoveryDocumentTemplate, bucketURL, bucketURL)
}
