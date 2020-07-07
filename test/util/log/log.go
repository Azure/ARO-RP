package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	logrus_test "github.com/sirupsen/logrus/hooks/test"
)

// ExpectedLogEntry contains a log message and log level which is expected to be
// emitted by the logging system.
type ExpectedLogEntry struct {
	// The message to be matched exactly. Conflicts with MessageRegex.
	Message string

	// The message to be matched as regex. Conflicts with Message.
	MessageRegex string

	// The logging level to be matched.
	Level logrus.Level
}

// assertMatches compares the expected entry with an actual Entry. An error is
// returned if they do not match.
func (ex ExpectedLogEntry) assertMatches(e logrus.Entry) string {
	if ex.Message != "" && ex.MessageRegex != "" {
		return "ExpectedLogEntry has both Message and MessageRegex set!"
	}

	if e.Level != ex.Level {
		return fmt.Sprintf("level: found %s, expected %s", e.Level, ex.Level)
	}

	if ex.Message != "" {
		if e.Message != ex.Message {
			return fmt.Sprintf("message: found `%s`, expected `%s`", e.Message, ex.Message)
		}

		return ""

	} else if ex.MessageRegex != "" {
		matched, err := regexp.MatchString(ex.MessageRegex, e.Message)
		if err != nil {
			return err.Error()
		}
		if matched != true {
			return fmt.Sprintf("message: found `%s`, expected to match `%s`", e.Message, ex.MessageRegex)
		}

		return ""

	} else {
		return fmt.Sprintf("ExpectedLogEntry has neither Message or MessageRegex set!")
	}

}

// NewCapturingLogger creates a logging hook and entry suitable for passing to
// functions and asserting on.
func NewCapturingLogger() (*logrus_test.Hook, *logrus.Entry) {
	logger, h := logrus_test.NewNullLogger()
	log := logrus.NewEntry(logger)
	return h, log
}

// AssertLoggingOutput compares the logs on `h` with the expected entries in
// `expected`. It returns a slice of errors encountered, with a zero length if
// no assertions failed.
func AssertLoggingOutput(h *logrus_test.Hook, expected []ExpectedLogEntry) []error {
	// We might need up to h.Entries errors, so just allocate as a block
	errs := make([]error, 0, len(h.Entries))

	if len(h.Entries) != len(expected) {
		errs = append(errs, fmt.Errorf("Got %d logs, expected %d", len(h.Entries), len(expected)))
	} else {
		for i, e := range h.Entries {
			errText := expected[i].assertMatches(e)
			if errText != "" {
				errs = append(errs, errors.Errorf("log #%d - %s", i, errText))
			}
		}
	}
	return errs
}
