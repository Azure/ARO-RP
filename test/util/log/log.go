package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

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
	audit.AddHook(logger)
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
	if len(expected) != len(h.AllEntries()) {
		t.Errorf("mismatch entries count: expected: %d, actual: %d", len(expected), len(h.AllEntries()))
		return
	}

	for i, entry := range h.AllEntries() {
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

		errors := []string{}
		if expected[i].EnvVer != actual.EnvVer {
			errors = append(errors, fmt.Sprintf("mismatch EnvVer. expected: %.f, actual: %.f", expected[i].EnvVer, actual.EnvVer))
		}

		if expected[i].EnvName != actual.EnvName {
			errors = append(errors, fmt.Sprintf("mismatch EnvName. expected: %s, actual: %s", expected[i].EnvName, actual.EnvName))
		}

		if expected[i].EnvFlags != actual.EnvFlags {
			errors = append(errors, fmt.Sprintf("mismatch EnvFlags. expected: %d, actual: %d", expected[i].EnvFlags, actual.EnvFlags))
		}

		if expected[i].EnvAppID != actual.EnvAppID {
			errors = append(errors, fmt.Sprintf("mismatch EnvAppID. expected: %s, actual: %s", expected[i].EnvAppID, actual.EnvAppID))
		}

		if expected[i].EnvCloudName != actual.EnvCloudName {
			errors = append(errors, fmt.Sprintf("mismatch EnvCloudName. expected: %s, actual: %s", expected[i].EnvCloudName, actual.EnvCloudName))
		}

		if expected[i].EnvCloudRole != actual.EnvCloudRole {
			errors = append(errors, fmt.Sprintf("mismatch EnvCloudRole. expected: %s, actual: %s", expected[i].EnvCloudRole, actual.EnvCloudRole))
		}

		if expected[i].EnvCloudRoleInstance != actual.EnvCloudRoleInstance {
			errors = append(errors, fmt.Sprintf("mismatch EnvCloudRoleInstance. expected: %s, actual: %s", expected[i].EnvCloudRoleInstance, actual.EnvCloudRoleInstance))
		}

		if expected[i].EnvCloudEnvironment != actual.EnvCloudEnvironment {
			errors = append(errors, fmt.Sprintf("mismatch EnvCloudEnvironment. expected: %s, actual: %s", expected[i].EnvCloudEnvironment, actual.EnvCloudEnvironment))
		}

		if expected[i].EnvCloudLocation != actual.EnvCloudLocation {
			errors = append(errors, fmt.Sprintf("mismatch EnvCloudLocation. expected: %s, actual: %s", expected[i].EnvCloudLocation, actual.EnvCloudLocation))
		}

		if expected[i].EnvCloudVer != actual.EnvCloudVer {
			errors = append(errors, fmt.Sprintf("mismatch EnvCloudVer. expected: %.f, actual: %.f", expected[i].EnvCloudVer, actual.EnvCloudVer))
		}

		if !reflect.DeepEqual(expected[i].CallerIdentities, actual.CallerIdentities) {
			errors = append(errors, fmt.Sprintf("mismatch CallerIdentities. expected: %+v, actual: %+v", expected[i].CallerIdentities, actual.CallerIdentities))
		}

		if expected[i].Category != actual.Category {
			errors = append(errors, fmt.Sprintf("mismatch Category. expected: %s, actual: %s", expected[i].Category, actual.Category))
		}

		if expected[i].OperationName != actual.OperationName {
			errors = append(errors, fmt.Sprintf("mismatch OperationName. expected: %s, actual: %s", expected[i].OperationName, actual.OperationName))
		}

		if !reflect.DeepEqual(expected[i].Result, actual.Result) {
			errors = append(errors, fmt.Sprintf("mismatch Result. expected: %+v, actual: %+v", expected[i].Result, actual.Result))
		}

		if !reflect.DeepEqual(expected[i].TargetResources, actual.TargetResources) {
			errors = append(errors, fmt.Sprintf("mismatch TargetResources. expected: %+v, actual: %+v", expected[i].TargetResources, actual.TargetResources))
		}

		if len(errors) > 0 {
			t.Errorf("mismatch payload fields: %s", strings.Join(errors, "\n"))
		}
	}
}
