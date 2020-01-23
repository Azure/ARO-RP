package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	encrypt "github.com/Azure/ARO-RP/pkg/util/encryption"
)

var _ codec.InterfaceExt = (*SecureBytesExt)(nil)

type SecureBytesExt struct {
	Cipher encryption.Cipher
}

func (s SecureBytesExt) ConvertExt(v interface{}) interface{} {
	data := v.(api.SecureBytes)
	if s.Cipher != nil {
		encrypted, err := s.Cipher.Encrypt(string(data))
		if err != nil {
			panic(err)
		}
		return encrypted
	}
	return string(data)
}
func (s SecureBytesExt) UpdateExt(dest interface{}, v interface{}) {
	output := dest.(*api.SecureBytes)
	if s.Cipher != nil {
		decrypted, err := s.Cipher.Decrypt(v.(string))
		if err != nil {
			panic(err)
		}
		*output = api.SecureBytes(decrypted)
		return
	}
	*output = api.SecureBytes(v.(string))
}

var _ codec.InterfaceExt = (*SecureStringExt)(nil)

type SecureStringExt struct {
	Cipher encrypt.Cipher
}

func (s SecureStringExt) ConvertExt(v interface{}) interface{} {
	data := v.(api.SecureString)
	if s.Cipher != nil {
		encrypted, err := s.Cipher.Encrypt(string(data))
		if err != nil {
			panic(err)
		}
		return encrypted
	}
	return string(data)
}
func (s SecureStringExt) UpdateExt(dest interface{}, v interface{}) {
	output := dest.(*api.SecureString)
	if s.Cipher != nil {
		decrypted, err := s.Cipher.Decrypt(v.(string))
		if err != nil {
			panic(err)
		}
		*output = api.SecureString(decrypted)
		return
	}
	*output = api.SecureString(v.(string))
}
