// stringifying representations of API documents for debugging and testing
// logging

package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"

	"github.com/ugorji/go/codec"
)

var secretPreservingJSONHandle *codec.JsonHandle

func init() {
	secretPreservingJSONHandle = &codec.JsonHandle{}

	err := secretPreservingJSONHandle.SetInterfaceExt(reflect.TypeFor[SecureBytes](), 1, secureHidingExt{})
	if err != nil {
		panic(err)
	}

	err = secretPreservingJSONHandle.SetInterfaceExt(reflect.TypeFor[*SecureString](), 1, secureHidingExt{})
	if err != nil {
		panic(err)
	}
}

func encodeJSON(i any) string {
	var b []byte

	err := codec.NewEncoderBytes(&b, secretPreservingJSONHandle).Encode(i)
	if err != nil {
		return err.Error()
	}

	return string(b)
}

var _ codec.InterfaceExt = (*secureHidingExt)(nil)

type secureHidingExt struct{}

func (secureHidingExt) ConvertExt(v any) any {
	return "[REDACTED]"
}

func (secureHidingExt) UpdateExt(dest any, v any) {
	panic("cannot be used to decode!")
}
