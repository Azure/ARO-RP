package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

type multi struct {
	sealer  AEAD
	openers []AEAD
}

var _ AEAD = (*multi)(nil)

func NewMulti(ctx context.Context, serviceKeyvault keyvault.Manager, secretName, legacySecretName string) (AEAD, error) {
	key, err := serviceKeyvault.GetBase64Secret(ctx, secretName, "")
	if err != nil {
		return nil, err
	}

	aead, err := NewAES256SHA512(ctx, key)
	if err != nil {
		return nil, err
	}

	m := &multi{
		sealer: aead,
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

func (c *multi) Open(input []byte) ([]byte, error) {
	for _, opener := range c.openers {
		b, err := opener.Open(input)
		if err == nil {
			return b, nil
		}
	}

	return nil, fmt.Errorf("could not open")
}

func (c *multi) Seal(input []byte) ([]byte, error) {
	return c.sealer.Seal(input)
}
