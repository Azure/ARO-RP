package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

type testStruct struct {
	SecureBytes  api.SecureBytes  `json:"secureBytes,omitempty"`
	SecureString api.SecureString `json:"secureString,omitempty"`
	Bytes        []byte           `json:"bytes,omitempty"`
	String       string           `json:"string,omitempty"`
}

func TestExtensions(t *testing.T) {
	ctx := context.Background()

	aead, err := encryption.NewXChaCha20Poly1305(ctx, []byte("\x63\xb5\x59\xf0\x43\x34\x79\x49\x68\x46\xab\x8b\xce\xdb\xc1\x2d\x7a\x0b\x14\x86\x7e\x1a\xb2\xd7\x3a\x92\x4e\x98\x6c\x5e\xcb\xe1"))
	if err != nil {
		t.Fatal(err)
	}

	h, err := NewJSONHandle(aead)
	if err != nil {
		t.Fatal(err)
	}

	encrypted := []byte(`{
		"bytes": "Ynl0ZXM=",
		"secureBytes": "6w8Uah0zX40LRfkYHuU9UvLuGrBcHb7l8I2M6qTcmtclOGJNONfHqAuaJWifZj7dd8fI",
		"secureString": "YT+ZNR23JBvILGw1WBn6/NhtCj9LM14EXp5VR6XloD7CN1MfmvW5FEn9duRSPYbdr98tLQ==",
		"string": "string"
	}`)

	decrypted := &testStruct{
		SecureBytes:  []byte("securebytes"),
		SecureString: "securestring",
		Bytes:        []byte("bytes"),
		String:       "string",
	}

	var ts testStruct
	err = codec.NewDecoderBytes(encrypted, h).Decode(&ts)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(&ts, decrypted) {
		t.Errorf("%#v", &ts)
	}

	var enc []byte
	err = codec.NewEncoderBytes(&enc, h).Encode(ts)
	if err != nil {
		t.Fatal(err)
	}

	ts = testStruct{}
	err = codec.NewDecoderBytes(encrypted, h).Decode(&ts)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(&ts, decrypted) {
		t.Errorf("%#v", &ts)
	}
}
