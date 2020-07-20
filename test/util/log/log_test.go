package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestAssertLoggingOutput(t *testing.T) {
	for _, tt := range []struct {
		name           string
		expectedLogs   []ExpectedLogEntry
		performLogging func(*logrus.Entry)
		wantErrs       []string
	}{
		{
			name: "Single log matches",
			expectedLogs: []ExpectedLogEntry{
				{
					Message: "Bar!",
					Level:   logrus.InfoLevel,
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
			},
		},
		{
			name: "Single log regex matches",
			expectedLogs: []ExpectedLogEntry{
				{
					MessageRegex: "B.*",
					Level:        logrus.InfoLevel,
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
			},
		},
		{
			name: "Multiple log matches",
			expectedLogs: []ExpectedLogEntry{
				{
					Message: "Bar!",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "Baz!",
					Level:   logrus.ErrorLevel,
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
				e.Error("Baz!")
			},
		},
		{
			name: "Log length miscount returns error",
			expectedLogs: []ExpectedLogEntry{
				{
					Message: "Bar!",
					Level:   logrus.InfoLevel,
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
				e.Error("Baz!")
			},
			wantErrs: []string{
				"Got 2 logs, expected 1",
				"--- emitted logs ---",
				"level: info, log text: Bar!",
				"level: error, log text: Baz!",
			},
		},
		{
			name: "Log level mismatch returns error",
			expectedLogs: []ExpectedLogEntry{
				{
					Message: "Bar!",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "Baz!",
					Level:   logrus.InfoLevel,
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
				e.Error("Baz!")
			},
			wantErrs: []string{
				"log #1 - level: found error, expected info",
			},
		},
		{
			name: "Log text mismatch returns error",
			expectedLogs: []ExpectedLogEntry{
				{
					Message: "Bar!",
					Level:   logrus.InfoLevel,
				},
				{
					Message: "Bar!",
					Level:   logrus.ErrorLevel,
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
				e.Error("Baz!")
			},
			wantErrs: []string{
				"log #1 - message: found `Baz!`, expected `Bar!`",
			},
		},
		{
			name: "Regex mismatch returns error",
			expectedLogs: []ExpectedLogEntry{
				{
					Message: "Bar!",
					Level:   logrus.InfoLevel,
				},
				{
					MessageRegex: "B.*",
					Level:        logrus.ErrorLevel,
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
				e.Error("Hat!")
			},
			wantErrs: []string{
				"log #1 - message: found `Hat!`, expected to match `B.*`",
			},
		},
		{
			name: "Bad regex returns error",
			expectedLogs: []ExpectedLogEntry{
				{
					MessageRegex: "[",
					Level:        logrus.ErrorLevel,
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Error("Hat!")
			},
			wantErrs: []string{
				"log #0 - error parsing regexp: missing closing ]: `[`",
			},
		},
		{
			name: "ExpectedLogEntry with no message returns an error",
			expectedLogs: []ExpectedLogEntry{
				{
					Level: logrus.ErrorLevel,
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Error("Hat!")
			},
			wantErrs: []string{
				"log #0 - ExpectedLogEntry has neither Message or MessageRegex set!",
			},
		},
		{
			name: "ExpectedLogEntry with both Message and MessageRegex returns an error",
			expectedLogs: []ExpectedLogEntry{
				{
					Message:      "foo",
					MessageRegex: "Baz",
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Error("Hat!")
			},
			wantErrs: []string{
				"log #0 - ExpectedLogEntry has both Message and MessageRegex set!",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, log := NewCapturingLogger()
			tt.performLogging(log)
			err := AssertLoggingOutput(h, tt.expectedLogs)

			if len(err) == len(tt.wantErrs) {
				for i, e := range err {
					if e.Error() != tt.wantErrs[i] {
						t.Error(tt.wantErrs[i], "!=", e)
					}
				}
			} else {
				t.Error("Different amount of errors returned than expected", err, tt.wantErrs)
			}
		})
	}
}
