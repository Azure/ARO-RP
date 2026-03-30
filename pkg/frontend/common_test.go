package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestLimitedWriter(t *testing.T) {
	for _, tt := range []struct {
		name         string
		writes       []string
		wantOutput   string
		wantExceeded bool
	}{
		{
			name:       "small write well under limit passes through unchanged",
			writes:     []string{"hello"},
			wantOutput: "hello",
		},
		{
			name:       "multiple small writes are concatenated without truncation",
			writes:     []string{"foo", "bar", "baz"},
			wantOutput: "foobarbaz",
		},
		{
			name: "write that exactly fills the limit is not truncated",
			writes: []string{
				strings.Repeat("x", int(execOutputLimit)),
			},
			wantOutput:   strings.Repeat("x", int(execOutputLimit)),
			wantExceeded: false,
		},
		{
			name: "write that exceeds the limit is truncated and notice is emitted",
			writes: []string{
				strings.Repeat("x", int(execOutputLimit)+10),
			},
			wantOutput:   strings.Repeat("x", int(execOutputLimit)) + "\n[stdout truncated at 1 MiB]\n",
			wantExceeded: true,
		},
		{
			name: "writes after limit is hit are silently dropped",
			writes: []string{
				strings.Repeat("x", int(execOutputLimit)+1),
				"this should not appear",
			},
			wantOutput:   strings.Repeat("x", int(execOutputLimit)) + "\n[stdout truncated at 1 MiB]\n",
			wantExceeded: true,
		},
		{
			name: "write that fills limit exactly then next write triggers truncation notice",
			writes: []string{
				strings.Repeat("x", int(execOutputLimit)),
				"overflow",
			},
			// The second write hits n==0 on entry: notice is emitted, "overflow" is dropped.
			wantOutput:   strings.Repeat("x", int(execOutputLimit)) + "\n[stdout truncated at 1 MiB]\n",
			wantExceeded: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			lw := newLimitedWriter(&buf, "stdout", logrus.NewEntry(logrus.New()))

			for _, w := range tt.writes {
				n, err := lw.Write([]byte(w))
				if err != nil {
					t.Fatalf("unexpected error from Write: %v", err)
				}
				if n != len(w) {
					t.Fatalf("Write returned %d, want %d (limitedWriter must always return len(p))", n, len(w))
				}
			}

			got := buf.String()
			if got != tt.wantOutput {
				t.Errorf("output mismatch:\ngot  %q\nwant %q", truncateForDisplay(got, 80), truncateForDisplay(tt.wantOutput, 80))
			}
			if lw.exceeded != tt.wantExceeded {
				t.Errorf("exceeded = %v, want %v", lw.exceeded, tt.wantExceeded)
			}
		})
	}
}

func TestLimitedWriter_UnderlyingWriterError(t *testing.T) {
	wantErr := errors.New("disk full")
	ew := &errWriter{err: wantErr}
	lw := newLimitedWriter(ew, "stdout", logrus.NewEntry(logrus.New()))

	n, err := lw.Write([]byte("hello"))
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v; want %v", err, wantErr)
	}
	if n != 0 {
		t.Errorf("n = %d; want 0 (errWriter returns 0)", n)
	}
}

type errWriter struct{ err error }

func (e *errWriter) Write(_ []byte) (int, error) { return 0, e.err }

// testWriteCloser is a WriteCloser backed by a bytes.Buffer for testing.
type testWriteCloser struct {
	*bytes.Buffer
	closed bool
}

func (w *testWriteCloser) Close() error {
	w.closed = true
	return nil
}

// truncateForDisplay shortens long strings for test error output.
func truncateForDisplay(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return fmt.Sprintf("%s... (total %d bytes)", s[:n], len(s))
}
