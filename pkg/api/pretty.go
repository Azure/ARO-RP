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

	err := secretPreservingJSONHandle.SetInterfaceExt(reflect.TypeOf(SecureBytes{}), 1, secureHidingExt{})
	if err != nil {
		panic(err)
	}

	err = secretPreservingJSONHandle.SetInterfaceExt(reflect.TypeOf((*SecureString)(nil)), 1, secureHidingExt{})
	if err != nil {
		panic(err)
	}
}

func encodeJSON(i interface{}) string {
	var b []byte

	err := codec.NewEncoderBytes(&b, secretPreservingJSONHandle).Encode(i)
	if err != nil {
		return err.Error()
	}

	return string(b)
}

var _ codec.InterfaceExt = (*secureHidingExt)(nil)

type secureHidingExt struct{}

func (secureHidingExt) ConvertExt(v interface{}) interface{} {
	return "[REDACTED]"
}

func (secureHidingExt) UpdateExt(dest interface{}, v interface{}) {
	panic("cannot be used to decode!")
}
