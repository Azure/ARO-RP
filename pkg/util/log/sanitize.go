package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"

	"github.com/sirupsen/logrus"
)

// Sanitize strips characters from a string that could be used to forge or
// corrupt log entries when the string originates from user-controlled input
// (CRLF log injection).  Carriage returns and line feeds are removed first --
// these are the characters used to inject fake log lines, and removing them
// explicitly is also recognised by static analysis as a log-injection
// sanitizer barrier.  Any remaining ASCII control characters (except tab) are
// stripped as well, since terminals and log viewers can interpret them to
// spoof output.
func Sanitize(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")

	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

// sanitizeHook is a logrus hook that sanitizes the message and any string
// valued fields of every log entry.  It provides defence-in-depth against log
// injection for all log call sites -- including generated code that we cannot
// annotate directly -- by running before the emitting hooks.
type sanitizeHook struct{}

func (sanitizeHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (sanitizeHook) Fire(entry *logrus.Entry) error {
	entry.Message = Sanitize(entry.Message)

	for k, v := range entry.Data {
		switch v := v.(type) {
		case string:
			entry.Data[k] = Sanitize(v)
		case error:
			// logrus patterns such as WithError(err) store an error value;
			// downstream hooks (e.g. journald) serialise it via fmt.Sprint, so
			// an error message containing CR/LF could still forge log lines.
			//
			// n.b. we deliberately do not sanitize fmt.Stringer values here:
			// this hook runs before the audit PayloadHook, which type-asserts
			// its typed struct fields, and converting them to strings would
			// silently break audit emission.
			entry.Data[k] = Sanitize(v.Error())
		}
	}

	return nil
}
