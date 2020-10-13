package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"testing"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"
)

func TestAssertLoggingOutput(t *testing.T) {
	for _, tt := range []struct {
		name           string
		expectedLogs   []map[string]types.GomegaMatcher
		performLogging func(*logrus.Entry)
		wantErr        string
	}{
		{
			name: "Single log matches",
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("Bar!"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
			},
		},
		{
			name: "Single log regex matches",
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.MatchRegexp("B.*"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
			},
		},
		{
			name: "Multiple log matches",
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("Bar!"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("Baz!"),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
				e.Error("Baz!")
			},
		},
		{
			name: "Log length miscount returns error",
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("Bar!"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
				e.Error("Baz!")
			},
			wantErr: "got 2 logs, expected 1",
		},
		{
			name: "Log level mismatch returns error",
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("Bar!"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("Baz!"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
				e.Error("Baz!")
			},
			wantErr: "log 1, field level, Expected\n    <logrus.Level>: 2\nto equal\n    <logrus.Level>: 4",
		},
		{
			name: "Log text mismatch returns error",
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("Bar!"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.Equal("Bar!"),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
				e.Error("Baz!")
			},
			wantErr: "log 1, field msg, Expected\n    <string>: Baz!\nto equal\n    <string>: Bar!",
		},
		{
			name: "Regex mismatch returns error",
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.Equal("Bar!"),
					"level": gomega.Equal(logrus.InfoLevel),
				},
				{
					"msg":   gomega.MatchRegexp("B.*"),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Info("Bar!")
				e.Error("Hat!")
			},
			wantErr: "log 1, field msg, Expected\n    <string>: Hat!\nto match regular expression\n    <string>: B.*",
		},
		{
			name: "Bad regex returns error",
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"msg":   gomega.MatchRegexp("["),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Error("Hat!")
			},
			wantErr: "log 0, field msg, error RegExp match failed to compile with error:\n\terror parsing regexp: missing closing ]: `[`",
		},
		{
			name: "ExpectedLogEntry with no message returns an error",
			expectedLogs: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			},
			performLogging: func(e *logrus.Entry) {
				e.Error("Hat!")
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, log := New()

			tt.performLogging(log)

			err := AssertLoggingOutput(h, tt.expectedLogs)
			if err == nil {
				if tt.wantErr != "" {
					t.Error(err)
				}
			} else {
				gotErr := strings.SplitN(err.Error(), "\nproblems:\n", 2)[1]
				if gotErr != tt.wantErr {
					t.Errorf("%q", gotErr)
				}
			}
		})
	}
}
