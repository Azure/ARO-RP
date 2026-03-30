package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// execOutputLimit is the per-stream output cap for exec stdout/stderr and pod logs.
	execOutputLimit      int64 = 1 << 20
	execOutputLimitLabel       = "1 MiB"

	adminActionCleanupTimeout = 30 * time.Second
	// adminActionStreamTimeout bounds the lifetime of streaming admin-action goroutines.
	adminActionStreamTimeout = 30 * time.Minute
)

// kubeRetryBackoff is the shared retry backoff for transient k8s API errors (not parallel-test-safe).
var kubeRetryBackoff = wait.Backoff{Steps: 3, Duration: 2 * time.Second, Factor: 2.0, Jitter: 0.1}

// limitedWriter truncates writes to execOutputLimit, emitting a single notice at that point.
type limitedWriter struct {
	w        io.Writer
	log      *logrus.Entry
	n        int64
	label    string
	exceeded bool
}

func newLimitedWriter(w io.Writer, label string, log *logrus.Entry) *limitedWriter {
	return &limitedWriter{w: w, log: log, n: execOutputLimit, label: label}
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.exceeded {
		return len(p), nil
	}
	toWrite := p
	truncated := false
	if lw.n == 0 {
		lw.exceeded = true
		lw.log.WithField("stream", lw.label).Warn("stream output truncated")
		_, _ = fmt.Fprintf(lw.w, "\n[%s truncated at %s]\n", lw.label, execOutputLimitLabel)
		return len(p), nil
	}
	if int64(len(p)) > lw.n {
		toWrite = p[:lw.n]
		truncated = true
	}
	n, err := lw.w.Write(toWrite)
	lw.n -= int64(n)
	if err != nil {
		return n, err
	}
	if truncated && n == len(toWrite) {
		lw.exceeded = true
		lw.log.WithField("stream", lw.label).Warn("stream output truncated")
		_, _ = fmt.Fprintf(lw.w, "\n[%s truncated at %s]\n", lw.label, execOutputLimitLabel)
	}
	// Return len(p), not n: io.Writer contract allows reporting short writes only when err != nil.
	return len(p), nil
}
