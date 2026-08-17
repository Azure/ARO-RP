package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSanitize(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain string is unchanged",
			in:   "hello world",
			want: "hello world",
		},
		{
			name: "newlines are stripped",
			in:   "line one\nline two",
			want: "line oneline two",
		},
		{
			name: "carriage returns are stripped",
			in:   "line one\r\nline two",
			want: "line oneline two",
		},
		{
			name: "forged log line is neutralised",
			in:   "value\ntime=\"2024-01-01\" level=error msg=\"spoofed\"",
			want: "value" + "time=\"2024-01-01\" level=error msg=\"spoofed\"",
		},
		{
			name: "other control characters are stripped",
			in:   "a\x00b\x1bc\x7fd",
			want: "abcd",
		},
		{
			name: "tab is preserved",
			in:   "a\tb",
			want: "a\tb",
		},
		{
			name: "unicode is preserved",
			in:   "café ☃",
			want: "café ☃",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sanitize(tt.in); got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeHook(t *testing.T) {
	var buf bytes.Buffer

	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	logger.SetOutput(&buf)
	logger.AddHook(&sanitizeHook{})

	logger.WithField("field", "injected\nfield").Info("injected\nmessage")

	out := buf.String()
	if bytes.Contains([]byte(out), []byte("injected\nmessage")) {
		t.Errorf("hook did not sanitize message: %q", out)
	}
	if bytes.Contains([]byte(out), []byte("injected\nfield")) {
		t.Errorf("hook did not sanitize field: %q", out)
	}
	if !bytes.Contains([]byte(out), []byte("injectedmessage")) {
		t.Errorf("sanitized message missing from output: %q", out)
	}
	if !bytes.Contains([]byte(out), []byte("injectedfield")) {
		t.Errorf("sanitized field missing from output: %q", out)
	}
}

func TestSanitizeHookErrorField(t *testing.T) {
	var buf bytes.Buffer

	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	logger.SetOutput(&buf)
	logger.AddHook(&sanitizeHook{})

	// WithError stores an error-typed field; a CR/LF in its message must not
	// survive to the emitted log line.
	logger.WithError(errors.New("injected\nerror")).Info("boom")

	out := buf.String()
	if bytes.Contains([]byte(out), []byte("injected\nerror")) {
		t.Errorf("hook did not sanitize error field: %q", out)
	}
	if !bytes.Contains([]byte(out), []byte("injectederror")) {
		t.Errorf("sanitized error field missing from output: %q", out)
	}
}
