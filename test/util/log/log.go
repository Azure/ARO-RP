package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"

	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
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
func AssertLoggingOutput(h *test.Hook, expected []map[string]types.GomegaMatcher) error {
	var problems []string

	if len(h.Entries) != len(expected) {
		problems = append(problems, fmt.Sprintf("got %d logs, expected %d", len(h.Entries), len(expected)))
	} else {
		for i, m := range expected {
			for k, matcher := range m {
				v := h.Entries[i].Data[k]
				switch k {
				case "level":
					v = h.Entries[i].Level
				case "msg":
					v = h.Entries[i].Message
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
		entries := make([]string, 0, len(h.Entries))

		for _, entry := range h.Entries {
			b, _ := entry.Logger.Formatter.Format(&entry)
			entries = append(entries, string(b))
		}

		return fmt.Errorf("logging mismatch:\ngot:\n%s\nproblems:\n%s", strings.Join(entries, ""), strings.Join(problems, "\n"))
	}

	return nil
}
