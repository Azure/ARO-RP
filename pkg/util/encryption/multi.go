package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"path/filepath"

	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

type multi struct {
	sealer      AEAD
	openers     []AEAD
	sealVersion string
}

var _ AEAD = (*multi)(nil)

func NewMulti(ctx context.Context, serviceKeyvault keyvault.Manager, secretName, legacySecretName string) (AEAD, error) {
	secret, err := serviceKeyvault.GetSecret(ctx, secretName, "")
	if err != nil {
		return nil, err
	}

	key, err := base64.StdEncoding.DecodeString(*secret.Value)
	if err != nil {
		return nil, err
	}

	aead, err := NewAES256SHA512(ctx, key)
	if err != nil {
		return nil, err
	}

	m := &multi{
		sealer:      aead,
		sealVersion: filepath.Base(*secret.ID),
	}

	for _, x := range []struct {
		secretName  string
		aeadFactory func(context.Context, []byte) (AEAD, error)
	}{
		{secretName, NewAES256SHA512},
		{legacySecretName, NewXChaCha20Poly1305},
	} {
		keys, err := serviceKeyvault.GetBase64Secrets(ctx, x.secretName)
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			aead, err = x.aeadFactory(ctx, key)
			if err != nil {
				return nil, err
			}

			m.openers = append(m.openers, aead)
		}
	}

	return m, nil
}

func (c *multi) Open(input []byte) (b []byte, err error) {
	for _, opener := range c.openers {
		b, err = opener.Open(input)
		if err == nil {
			return
		}
	}

	return nil, err
}

func (c *multi) Seal(input []byte) ([]byte, error) {
	return c.sealer.Seal(input)
}

func (c *multi) SealSecretVersion() string {
	return c.sealVersion
}
