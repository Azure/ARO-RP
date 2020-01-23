package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

type testStruct struct {
	SecureBytes  api.SecureBytes
	SecureString api.SecureString
	Bytes        []byte
	Str          string
}

func TestExtensions(t *testing.T) {
	encryption.RandRead = func(b []byte) (n int, err error) {
		b = make([]byte, len(b))
		return len(b), nil
	}

	key := make([]byte, 32)
	keybase64 := make([]byte, base64.StdEncoding.EncodedLen(len(key)))
	base64.StdEncoding.Encode(keybase64, key)
	env := &env.Test{TestSecret: keybase64}

	cipher, err := encryption.NewCipher(context.Background(), env)
	if err != nil {
		t.Error(err)
	}

	for _, tt := range []struct {
		name        string
		input       func(input *testStruct)
		inputCodec  *codec.JsonHandle
		output      func(input *testStruct)
		outputCodec *codec.JsonHandle
	}{
		{
			name: "noop",
		},
		{
			name: "SecureByte - encrypt - decrypt",
			input: func(input *testStruct) {
				input.SecureBytes = []byte("test")
			},
			output: func(output *testStruct) {
				output.SecureBytes = []byte("test")
			},
		},
		{
			name: "SecureByte - encrypt - raw",
			input: func(input *testStruct) {
				input.SecureBytes = []byte("test")
			},
			inputCodec: newJSONHandle(cipher),
			output: func(output *testStruct) {
				output.SecureBytes = []byte("ENC*AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADPvl/edTVlZfXuNqdeWf2B1jR50=")
				// empty string encoded
				output.SecureString = "ENC*AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAjzuUWlGQbchgDen4li0A5g=="
			},
			outputCodec: newJSONHandle(nil),
		},
		{
			name: "SecureByte - raw - decrypt",
			input: func(input *testStruct) {
				input.SecureBytes = []byte("ENC*AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADPvl/edTVlZfXuNqdeWf2B1jR50=")
			},
			inputCodec: newJSONHandle(nil),
			output: func(output *testStruct) {
				output.SecureBytes = []byte("test")
			},
			outputCodec: newJSONHandle(cipher),
		},
		{
			name: "SecureByte - raw - raw",
			input: func(input *testStruct) {
				input.SecureBytes = []byte("ENC*AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADPvl/edTVlZfXuNqdeWf2B1jR50=")
			},
			inputCodec: newJSONHandle(nil),
			output: func(output *testStruct) {
				output.SecureBytes = []byte("ENC*AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADPvl/edTVlZfXuNqdeWf2B1jR50=")
			},
			outputCodec: newJSONHandle(nil),
		},
		{
			name: "SecureString - encrypt - decrypt",
			input: func(input *testStruct) {
				input.SecureString = "test"
			},
			output: func(output *testStruct) {
				output.SecureString = "test"
			},
		},
		{
			name: "SecureString - encrypt - raw",
			input: func(input *testStruct) {
				input.SecureString = "test"
			},
			inputCodec: newJSONHandle(cipher),
			output: func(output *testStruct) {
				output.SecureString = "ENC*AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADPvl/edTVlZfXuNqdeWf2B1jR50="
			},
			outputCodec: newJSONHandle(nil),
		},
		{
			name: "SecureString - raw - decrypt",
			input: func(input *testStruct) {
				input.SecureString = "ENC*AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADPvl/edTVlZfXuNqdeWf2B1jR50="
			},
			inputCodec: newJSONHandle(nil),
			output: func(output *testStruct) {
				output.SecureString = "test"
			},
			outputCodec: newJSONHandle(cipher),
		},
		{
			name: "SecureString - raw - raw",
			input: func(input *testStruct) {
				input.SecureString = "ENC*AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADPvl/edTVlZfXuNqdeWf2B1jR50="
			},
			inputCodec: newJSONHandle(nil),
			output: func(output *testStruct) {
				output.SecureString = "ENC*AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADPvl/edTVlZfXuNqdeWf2B1jR50="
			},
			outputCodec: newJSONHandle(nil),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.inputCodec == nil {
				tt.inputCodec = newJSONHandle(cipher)
			}
			if tt.outputCodec == nil {
				tt.outputCodec = newJSONHandle(cipher)
			}

			input := &testStruct{}
			if tt.input != nil {
				tt.input(input)
			}

			output := &testStruct{}
			if tt.output != nil {
				tt.output(output)
			}

			buf := &bytes.Buffer{}
			err = codec.NewEncoder(buf, tt.inputCodec).Encode(input)
			if err != nil {
				t.Error(err)
			}
			data, err := ioutil.ReadAll(buf)
			if err != nil {
				t.Error(err)
			}

			result := &testStruct{}
			err = codec.NewDecoder(bytes.NewReader(data), tt.outputCodec).Decode(result)
			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(output, result) {
				output, _ := json.Marshal(output)
				result, _ := json.Marshal(result)
				t.Errorf("\n wants: %s,'\ngot: %s", string(output), string(result))
			}
		})
	}
}
