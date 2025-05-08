package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
)

type multi struct {
	sealer  AEAD
	openers []AEAD
}

var _ AEAD = (*multi)(nil)

func NewMulti(ctx context.Context, serviceKeyvault azsecrets.Client, secretName, legacySecretName string) (AEAD, error) {
	rawKey, err := serviceKeyvault.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		return nil, err
	}

	key, err := azsecrets.ExtractBase64Value(rawKey)
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
		var keys [][]byte
		pager := serviceKeyvault.NewListSecretPropertiesVersionsPager(x.secretName, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, properties := range page.Value {
				if properties != nil && properties.ID != nil && properties.Attributes != nil {
					raw, err := serviceKeyvault.GetSecret(ctx, (*properties.ID).Name(), (*properties.ID).Version(), nil)
					if err != nil {
						return nil, err
					}
					version, err := azsecrets.ExtractBase64Value(raw)
					if err != nil {
						return nil, err
					}
					keys = append(keys, version)
				}
			}
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
