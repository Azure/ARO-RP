package responsewriter

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"net/http"
	"testing"
)

func TestResponseWriter(t *testing.T) {
	w := New(&http.Request{ProtoMajor: 1, ProtoMinor: 1})

	http.NotFound(w, nil)

	buf := &bytes.Buffer{}
	_ = w.Response().Write(buf)

	if buf.String() != "HTTP/1.1 404 Not Found\r\nConnection: close\r\nContent-Type: text/plain; charset=utf-8\r\nX-Content-Type-Options: nosniff\r\n\r\n404 page not found\n" {
		t.Error(buf.String())
	}
}
