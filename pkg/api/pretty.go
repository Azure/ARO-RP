// stringifying representations of API documents for debugging and testing
// logging

package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"strings"

	"github.com/ugorji/go/codec"
)

func newSecretPreservingJsonHandle() *codec.JsonHandle {
	h := &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				ErrorIfNoField: true,
			},
		},
	}

	h.SetInterfaceExt(reflect.TypeOf(SecureBytes{}), 1, secureHidingExt{})
	h.SetInterfaceExt(reflect.TypeOf((*SecureString)(nil)), 1, secureHidingExt{})
	return h
}

func encodeJSON(i interface{}) string {
	w := &strings.Builder{}
	enc := codec.NewEncoder(w, newSecretPreservingJsonHandle())
	err := enc.Encode(i)
	if err != nil {
		return err.Error()
	}
	return w.String()
}

var _ codec.InterfaceExt = (*secureHidingExt)(nil)

type secureHidingExt struct {
}

func (s secureHidingExt) ConvertExt(v interface{}) interface{} {
	return "[REDACTED]"
}

func (s secureHidingExt) UpdateExt(dest interface{}, v interface{}) {
	panic("cannot be used to decode!")
}
