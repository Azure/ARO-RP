package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

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

	aead, err := NewAES256SHA512(ctx, key, "")
	if err != nil {
		return nil, err
	}

	m := &multi{
		sealer: aead,
	}

	for _, x := range []struct {
		secretName  string
		aeadFactory func(context.Context, []byte, string) (AEAD, error)
	}{
		{secretName, NewAES256SHA512},
		{legacySecretName, NewXChaCha20Poly1305},
	} {
		pager := serviceKeyvault.NewListSecretPropertiesVersionsPager(x.secretName, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, properties := range page.Value {
				if properties != nil && properties.ID != nil && properties.Attributes != nil {
					secretVersion := (*properties.ID).Version()
					raw, err := serviceKeyvault.GetSecret(ctx, (*properties.ID).Name(), secretVersion, nil)
					if err != nil {
						return nil, err
					}
					version, err := azsecrets.ExtractBase64Value(raw)
					if err != nil {
						return nil, err
					}

					aead, err = x.aeadFactory(ctx, version, secretVersion)
					if err != nil {
						return nil, err
					}

					m.openers = append(m.openers, aead)
				}
			}
		}
	}

	return m, nil
}

func (c *multi) Name() string {
	var openerNames []string

	for _, i := range c.openers {
		openerNames = append(openerNames, i.Name())
	}

	return fmt.Sprintf("Multi(sealer=%s, openers=%s)", c.sealer.Name(), strings.Join(openerNames, ","))
}

func (c *multi) Open(input []byte) ([]byte, error) {
	var errs []error

	for _, opener := range c.openers {
		b, err := opener.Open(input)
		if err == nil {
			return b, nil
		} else {
			errs = append(errs, fmt.Errorf("%s: %w", opener.Name(), err))
		}
	}

	errStrings := ""
	for _, err := range errs {
		errStrings = errStrings + fmt.Sprintf("\n\t* %s", err)
	}

	return nil, fmt.Errorf("no openers succeeded:%s", errStrings)
}

func (c *multi) Seal(input []byte) ([]byte, error) {
	return c.sealer.Seal(input)
}
