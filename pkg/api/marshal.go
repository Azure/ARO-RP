package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"reflect"

	"github.com/ugorji/go/codec"
)

// AddExtensions adds extensions to a ugorji/go/codec to enable it to serialise
// our types properly. If cipher is provided, it will encrypt/decrypt sensitive fields
func AddExtensions(h *codec.BasicHandle) error {
	err := h.AddExt(reflect.TypeOf(&rsa.PrivateKey{}), 0, func(v reflect.Value) ([]byte, error) {
		if reflect.DeepEqual(v.Elem().Interface(), rsa.PrivateKey{}) {
			return nil, nil
		}
		data := x509.MarshalPKCS1PrivateKey(v.Interface().(*rsa.PrivateKey))

		return data, nil
	}, func(v reflect.Value, b []byte) error {
		key, err := x509.ParsePKCS1PrivateKey(b)
		if err != nil {
			return err
		}
		v.Elem().Set(reflect.ValueOf(key).Elem())
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// MarshalJSON marshals an InstallPhase
func (p InstallPhase) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// UnmarshalJSON unmarshals an InstallPhase
func (p *InstallPhase) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*p, err = InstallPhaseString(s)
	return err
}
