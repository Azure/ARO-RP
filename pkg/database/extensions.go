package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"

	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

var _ codec.InterfaceExt = (*secureBytesExt)(nil)

type secureBytesExt struct {
	aead encryption.AEAD
}

func (s secureBytesExt) ConvertExt(v interface{}) interface{} {
	b, err := s.aead.Seal(v.(api.SecureBytes))
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(b)
}

func (s secureBytesExt) UpdateExt(dest interface{}, v interface{}) {
	b, err := base64.StdEncoding.DecodeString(v.(string))
	if err != nil {
		panic(err)
	}

	b, err = s.aead.Open(b)
	if err != nil {
		panic(err)
	}

	*dest.(*api.SecureBytes) = b
}

var _ codec.InterfaceExt = (*secureStringExt)(nil)

type secureStringExt struct {
	aead encryption.AEAD
}

func (s secureStringExt) ConvertExt(v interface{}) interface{} {
	b, err := s.aead.Seal([]byte(v.(api.SecureString)))
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(b)
}

func (s secureStringExt) UpdateExt(dest interface{}, v interface{}) {
	b, err := base64.StdEncoding.DecodeString(v.(string))
	if err != nil {
		panic(err)
	}

	b, err = s.aead.Open(b)
	if err != nil {
		panic(err)
	}

	*dest.(*api.SecureString) = api.SecureString(b)
}
