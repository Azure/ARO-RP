package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"testing"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Azure/ARO-RP/pkg/util/log/audit"
)

// NewAudit creates a logging hook and entry suitable for testing the IFXAudit
// feature.
func NewAudit() (*test.Hook, *logrus.Entry) {
	logger, h := test.NewNullLogger()
	logger.AddHook(&audit.PayloadHook{
		Payload: &audit.Payload{},
	})
	return h, logrus.NewEntry(logger)
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
