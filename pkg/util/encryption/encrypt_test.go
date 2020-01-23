package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/env"
)

func TestEncryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	keybase64 := make([]byte, base64.StdEncoding.EncodedLen(len(key)))
	base64.StdEncoding.Encode(keybase64, key)
	env := &env.Test{TestSecret: keybase64}

	cipher, err := NewCipher(context.Background(), env)
	if err != nil {
		t.Error(err)
	}

	test := "secert"
	encrypted, err := cipher.Encrypt(test)
	if err != nil {
		t.Error(err)
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Error(err)
	}

	if r := strings.Compare(test, decrypted); r != 0 {
		t.Error("encryption roundTrip failed")
	}
}

func TestEncrypt(t *testing.T) {
	RandRead = func(b []byte) (n int, err error) {
		b = make([]byte, len(b))
		return len(b), nil
	}

	for _, tt := range []struct {
		name     string
		input    string
		expected string
		wantErr  string
		env      func(e *env.Test)
	}{
		{
			name:     "ok encrypt",
			input:    "test",
			expected: "ENC*AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADPvl/edTVlZfXuNqdeWf2B1jR50=",
			wantErr:  "",
			env: func(input *env.Test) {
				key := make([]byte, 32)
				keybase64 := make([]byte, base64.StdEncoding.EncodedLen(len(key)))
				base64.StdEncoding.Encode(keybase64, key)
				input.TestSecret = keybase64
			},
		},
		{
			name:    "base64 key error",
			wantErr: "illegal base64 data at input byte 8",
			env: func(input *env.Test) {
				input.TestSecret = []byte("badsecret")
			},
		},
		{
			name:     "key too short",
			input:    "test",
			expected: "ENC*AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADPvl/edTVlZfXuNqdeWf2B1jR50=",
			wantErr:  "chacha20poly1305: bad key length",
			env: func(input *env.Test) {
				keybase64 := base64.StdEncoding.EncodeToString(make([]byte, 15))
				input.TestSecret = []byte(keybase64)
			},
		},
		{
			name:     "key too long", // due to base64 approximations library truncates the secret to right lenhgt
			input:    "test",
			expected: "ENC*AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADPvl/edTVlZfXuNqdeWf2B1jR50=",
			env: func(input *env.Test) {
				keybase64 := base64.StdEncoding.EncodeToString((make([]byte, 40)))
				input.TestSecret = []byte(keybase64)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := &env.Test{}
			if tt.env != nil {
				tt.env(e)
			}

			cipher, err := NewCipher(context.Background(), e)
			if err != nil {
				if err.Error() != tt.wantErr {
					t.Errorf("\n wants: %s,'\ngot: %s", tt.wantErr, err.Error())
					t.FailNow()
				}
				t.SkipNow()
			}

			result, err := cipher.Encrypt(tt.input)
			if err != nil {
				t.Error(err)
			}
			if tt.expected != result {
				t.Errorf("\n wants: %s,'\ngot: %s", tt.expected, result)
			}
		})
	}
}
