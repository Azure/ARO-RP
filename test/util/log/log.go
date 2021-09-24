package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/go-test/deep"
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

// NewAudit creates a logging hook and entry suitable for testing the IFXAudit
// feature.
func NewAudit() (*test.Hook, *logrus.Entry) {
	logger, h := test.NewNullLogger()
	logger.AddHook(&audit.PayloadHook{
		Payload: &audit.Payload{},
	})
	return h, logrus.NewEntry(logger)
}

// AssertLoggingOutput compares the logs on `h` with the expected entries in
// `expected`. It returns a slice of errors encountered, with a zero length if
// no assertions failed.
func AssertLoggingOutput(h *test.Hook, expected []map[string]types.GomegaMatcher) error {
	var (
		problems []string
		entries  = h.Entries
	)

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
			b, _ := entry.Logger.Formatter.Format(&entry)
			formatted = append(formatted, string(b))
		}

		return fmt.Errorf("logging mismatch:\ngot:\n%s\nproblems:\n%s", strings.Join(formatted, ""), strings.Join(problems, "\n"))
	}

	return nil
}

// AssertAuditPayload compares the audit payloads in `h` with the given expected
// payloads
func AssertAuditPayloads(t *testing.T, h *test.Hook, expected []*audit.Payload) {
	actualEntries := []*audit.Payload{}

	for _, entry := range h.AllEntries() {
		raw, ok := entry.Data[audit.MetadataPayload].(string)
		if !ok {
			t.Error("audit payload type cast failed")
			return
		}

		var actual audit.Payload
		if err := json.Unmarshal([]byte(raw), &actual); err != nil {
			t.Errorf("fail to unmarshal payload: %s", err)
			continue
		}
		actualEntries = append(actualEntries, &actual)
	}

	if len(expected) == 0 && len(actualEntries) == 0 {
		return
	}

	r := deep.Equal(expected, actualEntries)
	if len(r) != 0 {
		t.Error("log differences -- expected - actual:")
		for _, entry := range r {
			t.Error(entry)
		}
	}
}
