package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"io"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// execOutputLimit is the per-stream output cap for exec stdout/stderr and pod logs.
	execOutputLimit      int64  = 1 << 20
	execOutputLimitLabel string = "1 MiB" // must match execOutputLimit

	adminActionCleanupTimeout = 30 * time.Second
)

// kubeRetryBackoff is the shared retry backoff for transient Kubernetes API errors.
var kubeRetryBackoff = wait.Backoff{Steps: 3, Duration: 2 * time.Second, Factor: 1.0}

// limitedWriter truncates writes to execOutputLimit, emitting a single notice at that point.
type limitedWriter struct {
	w        io.Writer
	n        int64
	label    string
	exceeded bool
}

func newLimitedWriter(w io.Writer, label string) *limitedWriter {
	return &limitedWriter{w: w, n: execOutputLimit, label: label}
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.exceeded {
		return len(p), nil
	}
	toWrite := p
	truncated := false
	if lw.n == 0 {
		lw.exceeded = true
		_, _ = fmt.Fprintf(lw.w, "\n[%s truncated at %s]\n", lw.label, execOutputLimitLabel)
		return len(p), nil
	}
	if int64(len(p)) > lw.n {
		toWrite = p[:lw.n]
		truncated = true
	}
	n, err := lw.w.Write(toWrite)
	lw.n -= int64(n)
	if truncated && err == nil && n == len(toWrite) {
		lw.exceeded = true
		_, _ = fmt.Fprintf(lw.w, "\n[%s truncated at %s]\n", lw.label, execOutputLimitLabel)
	}
	return len(p), err
}

// nopWriteCloser wraps an io.Writer with a no-op Close for use with functions that close their writer.
type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }
