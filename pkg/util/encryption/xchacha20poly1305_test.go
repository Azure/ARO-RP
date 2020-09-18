package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/env"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestNewXChaCha20Poly1305(t *testing.T) {
	for _, tt := range []struct {
		name    string
		key     []byte
		wantErr string
	}{
		{
			name: "valid",
			key:  []byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
		},
		{
			name:    "key too short",
			key:     []byte("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"),
			wantErr: "chacha20poly1305: bad key length",
		},
		{
			name:    "key too long",
			key:     []byte("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"),
			wantErr: "chacha20poly1305: bad key length",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().GetBase64Secret(gomock.Any(), env.EncryptionSecretName).Return(tt.key, nil)

			_, err := NewXChaCha20Poly1305(context.Background(), _env, env.EncryptionSecretName)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Fatal(err)
			}
		})
	}
}

func TestXChaCha20Poly1305Decrypt(t *testing.T) {
	for _, tt := range []struct {
		name          string
		key           []byte
		input         []byte
		wantDecrypted []byte
		wantErr       string
	}{
		{
			name:          "valid",
			key:           []byte("\x6a\x98\x95\x6b\x2b\xb2\x7e\xfd\x1b\x68\xdf\x5c\x40\xc3\x4f\x8b\xcf\xff\xe8\x17\xc2\x2d\xf6\x40\x2e\x5a\xb0\x15\x63\x4a\x2d\x2e"),
			input:         []byte("\xd9\x1c\x3c\x05\xb2\xf3\xc5\x93\x20\x9f\x9b\x67\x43\x8c\x0c\x3d\x9c\x33\x5b\x16\xd6\x9a\x9c\xf2\x9c\xf6\xe9\xbd\xdd\xe3\x1d\x54\xde\x41\xa2\x99\x56\x6a\xfc\x9a\xf3\x58\x73\x03"),
			wantDecrypted: []byte("test"),
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
			input:   []byte("XXXXXXXXXXXXXXXXXXXXXXX"),
			wantErr: "encrypted value too short",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().GetBase64Secret(gomock.Any(), env.EncryptionSecretName).Return(tt.key, nil)

			cipher, err := NewXChaCha20Poly1305(context.Background(), _env, env.EncryptionSecretName)
			if err != nil {
				t.Fatal(err)
			}

			decrypted, err := cipher.Decrypt(tt.input)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Fatal(err)
			}

			if !bytes.Equal(tt.wantDecrypted, decrypted) {
				t.Error(string(decrypted))
			}
		})
	}
}

func TestXChaCha20Poly1305Encrypt(t *testing.T) {
	for _, tt := range []struct {
		name          string
		key           []byte
		randRead      func(b []byte) (int, error)
		input         []byte
		wantEncrypted []byte
		wantErr       string
	}{
		{
			name: "valid",
			key:  []byte("\x6a\x98\x95\x6b\x2b\xb2\x7e\xfd\x1b\x68\xdf\x5c\x40\xc3\x4f\x8b\xcf\xff\xe8\x17\xc2\x2d\xf6\x40\x2e\x5a\xb0\x15\x63\x4a\x2d\x2e"),
			randRead: func(b []byte) (int, error) {
				nonce := []byte("\xd9\x1c\x3c\x05\xb2\xf3\xc5\x93\x20\x9f\x9b\x67\x43\x8c\x0c\x3d\x9c\x33\x5b\x16\xd6\x9a\x9c\xf2")
				copy(b, nonce)
				return len(nonce), nil
			},
			input:         []byte("test"),
			wantEncrypted: []byte("\xd9\x1c\x3c\x05\xb2\xf3\xc5\x93\x20\x9f\x9b\x67\x43\x8c\x0c\x3d\x9c\x33\x5b\x16\xd6\x9a\x9c\xf2\x9c\xf6\xe9\xbd\xdd\xe3\x1d\x54\xde\x41\xa2\x99\x56\x6a\xfc\x9a\xf3\x58\x73\x03"),
		},
		{
			name: "rand.Read error",
			key:  []byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
			randRead: func(b []byte) (int, error) {
				return 0, fmt.Errorf("random error")
			},
			wantErr: "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().GetBase64Secret(gomock.Any(), env.EncryptionSecretName).Return(tt.key, nil)

			cipher, err := NewXChaCha20Poly1305(context.Background(), _env, env.EncryptionSecretName)
			if err != nil {
				t.Fatal(err)
			}

			cipher.(*aeadCipher).randRead = tt.randRead

			encrypted, err := cipher.Encrypt(tt.input)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Fatal(err)
			}

			if !bytes.Equal(tt.wantEncrypted, encrypted) {
				t.Error(hex.EncodeToString(encrypted))
			}
		})
	}
}
