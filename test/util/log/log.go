package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"

	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Azure/ARO-RP/pkg/util/log/audit"
)

// New creates a logging hook and entry suitable for passing to functions and
// asserting on.
func New() (*test.Hook, *logrus.Entry) {
	logger, h := test.NewNullLogger()
	log := logrus.NewEntry(logger)
	return h, log
}

// AssertLoggingOutput compares the logs on `h` with the expected entries in
// `expected`. It returns a slice of errors encountered, with a zero length if
// no assertions failed.
// This function skips audit log entries in `h` to avoid test flakiness. Use
// audit.AssertAuditingOutput() to verify audit entries.
func AssertLoggingOutput(h *test.Hook, expected []map[string]types.GomegaMatcher) error {
	var (
		problems []string
		entries  []*logrus.Entry
	)

	// skip audit entries to avoid test flakiness
	for _, e := range h.AllEntries() {
		if v := e.Data[audit.MetadataLogKind]; v != audit.IFXAuditLogKind {
			entries = append(entries, e)
		}
	}

	if len(entries) != len(expected) {
		problems = append(problems, fmt.Sprintf("got %d logs, expected %d", len(entries), len(expected)))
	} else {
		for i, m := range expected {
			for k, matcher := range m {
				v := entries[i].Data[k]
				switch k {
				case "level":
					v = entries[i].Level
				case "msg":
					v = entries[i].Message
				}
				ok, err := matcher.Match(v)
				if err != nil {
					problems = append(problems, fmt.Sprintf("log %d, field %s, error %s", i, k, err))
				} else if !ok {
					problems = append(problems, fmt.Sprintf("log %d, field %s, %s", i, k, matcher.FailureMessage(v)))
				}
			}
		}
	}

	if len(problems) > 0 {
		formatted := make([]string, 0, len(entries))

		for _, entry := range entries {
			b, _ := entry.Logger.Formatter.Format(entry)
			formatted = append(formatted, string(b))
		}

		return fmt.Errorf("logging mismatch:\ngot:\n%s\nproblems:\n%s", strings.Join(formatted, ""), strings.Join(problems, "\n"))
	}

	return nil
}
