package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"

	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestLog(t *testing.T) {
	h, log := testlog.New()

	ctx := context.WithValue(context.Background(), ContextKeyUsername, "username")
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://localhost/", strings.NewReader("body"))
	if err != nil {
		t.Fatal(err)
	}
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("User-Agent", "user-agent")

	w := httptest.NewRecorder()

	Log(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL = nil // mutate the request

		_ = w.(http.Hijacker) // must implement http.Hijacker

		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, r.Body)
	})).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Error(w.Code)
	}

	expected := []map[string]types.GomegaMatcher{
		{
			"msg":                 gomega.Equal("read request"),
			"level":               gomega.Equal(logrus.InfoLevel),
			"request_method":      gomega.Equal("POST"),
			"request_path":        gomega.Equal("/"),
			"request_proto":       gomega.Equal("HTTP/1.1"),
			"request_remote_addr": gomega.Equal("127.0.0.1:1234"),
			"request_user_agent":  gomega.Equal("user-agent"),
			"username":            gomega.Equal("username"),
		},
		{
			"msg":                  gomega.Equal("sent response"),
			"level":                gomega.Equal(logrus.InfoLevel),
			"body_read_bytes":      gomega.Equal(4),
			"body_written_bytes":   gomega.Equal(4),
			"response_status_code": gomega.Equal(http.StatusOK),
			"request_method":       gomega.Equal("POST"),
			"request_path":         gomega.Equal("/"),
			"request_proto":        gomega.Equal("HTTP/1.1"),
			"request_remote_addr":  gomega.Equal("127.0.0.1:1234"),
			"request_user_agent":   gomega.Equal("user-agent"),
			"username":             gomega.Equal("username"),
		},
	}

	err = testlog.AssertLoggingOutput(h, expected)
	if err != nil {
		t.Error(err)
	}

	for _, e := range h.Entries {
		fmt.Println(e)
	}
}
