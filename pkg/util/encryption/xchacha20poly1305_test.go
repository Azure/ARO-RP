package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"testing"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestNewXChaCha20Poly1305(t *testing.T) {
	for _, tt := range []struct {
		name    string
		key     []byte
		wantErr string
	}{
		{
			name: "valid",
			key:  make([]byte, 32),
		},
		{
			name:    "key too short",
			key:     make([]byte, 31),
			wantErr: "chacha20poly1305: bad key length",
		},
		{
			name:    "key too long",
			key:     make([]byte, 33),
			wantErr: "chacha20poly1305: bad key length",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewXChaCha20Poly1305(context.Background(), tt.key)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestXChaCha20Poly1305Open(t *testing.T) {
	for _, tt := range []struct {
		name       string
		key        []byte
		input      []byte
		wantOpened []byte
		wantErr    string
	}{
		{
			name:       "valid",
			key:        []byte("\x6a\x98\x95\x6b\x2b\xb2\x7e\xfd\x1b\x68\xdf\x5c\x40\xc3\x4f\x8b\xcf\xff\xe8\x17\xc2\x2d\xf6\x40\x2e\x5a\xb0\x15\x63\x4a\x2d\x2e"),
			input:      []byte("\xd9\x1c\x3c\x05\xb2\xf3\xc5\x93\x20\x9f\x9b\x67\x43\x8c\x0c\x3d\x9c\x33\x5b\x16\xd6\x9a\x9c\xf2\x9c\xf6\xe9\xbd\xdd\xe3\x1d\x54\xde\x41\xa2\x99\x56\x6a\xfc\x9a\xf3\x58\x73\x03"),
			wantOpened: []byte("test"),
		},
		{
			name:    "invalid - encrypted value tampered with",
			key:     []byte("\x6a\x98\x95\x6b\x2b\xb2\x7e\xfd\x1b\x68\xdf\x5c\x40\xc3\x4f\x8b\xcf\xff\xe8\x17\xc2\x2d\xf6\x40\x2e\x5a\xb0\x15\x63\x4a\x2d\x2e"),
			input:   []byte("\xda\x1c\x3c\x05\xb2\xf3\xc5\x93\x20\x9f\x9b\x67\x43\x8c\x0c\x3d\x9c\x33\x5b\x16\xd6\x9a\x9c\xf2\x9c\xf6\xe9\xbd\xdd\xe3\x1d\x54\xde\x41\xa2\x99\x56\x6a\xfc\x9a\xf3\x58\x73\x03"),
			wantErr: "chacha20poly1305: message authentication failed",
		},
		{
			name:    "invalid - too short",
			key:     []byte("\x6a\x98\x95\x6b\x2b\xb2\x7e\xfd\x1b\x68\xdf\x5c\x40\xc3\x4f\x8b\xcf\xff\xe8\x17\xc2\x2d\xf6\x40\x2e\x5a\xb0\x15\x63\x4a\x2d\x2e"),
			input:   make([]byte, 23),
			wantErr: "encrypted value too short",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			aead, err := NewXChaCha20Poly1305(context.Background(), tt.key)
			if err != nil {
				t.Fatal(err)
			}

			opened, err := aead.Open(tt.input)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if !bytes.Equal(tt.wantOpened, opened) {
				t.Error(string(opened))
			}
		})
	}
}

func TestXChaCha20Poly1305Seal(t *testing.T) {
	for _, tt := range []struct {
		name       string
		key        []byte
		randReader io.Reader
		input      []byte
		wantSealed []byte
		wantErr    string
	}{
		{
			name:       "valid",
			key:        []byte("\x6a\x98\x95\x6b\x2b\xb2\x7e\xfd\x1b\x68\xdf\x5c\x40\xc3\x4f\x8b\xcf\xff\xe8\x17\xc2\x2d\xf6\x40\x2e\x5a\xb0\x15\x63\x4a\x2d\x2e"),
			randReader: bytes.NewBufferString("\xd9\x1c\x3c\x05\xb2\xf3\xc5\x93\x20\x9f\x9b\x67\x43\x8c\x0c\x3d\x9c\x33\x5b\x16\xd6\x9a\x9c\xf2"),
			input:      []byte("test"),
			wantSealed: []byte("\xd9\x1c\x3c\x05\xb2\xf3\xc5\x93\x20\x9f\x9b\x67\x43\x8c\x0c\x3d\x9c\x33\x5b\x16\xd6\x9a\x9c\xf2\x9c\xf6\xe9\xbd\xdd\xe3\x1d\x54\xde\x41\xa2\x99\x56\x6a\xfc\x9a\xf3\x58\x73\x03"),
		},
		{
			name:       "rand.Read EOF",
			key:        make([]byte, 32),
			randReader: &bytes.Buffer{},
			wantErr:    "EOF",
		},
		{
			name:       "rand.Read unexpected EOF",
			key:        make([]byte, 32),
			randReader: bytes.NewBufferString("X"),
			wantErr:    "unexpected EOF",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			aead, err := NewXChaCha20Poly1305(context.Background(), tt.key)
			if err != nil {
				t.Fatal(err)
			}

			aead.(*xChaCha20Poly1305).randReader = tt.randReader

			sealed, err := aead.Seal(tt.input)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if !bytes.Equal(tt.wantSealed, sealed) {
				t.Error(hex.EncodeToString(sealed))
			}
		})
	}
}
