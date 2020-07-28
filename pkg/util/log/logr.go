package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
)

type logrWrapper struct {
	entry *logrus.Entry
	level int
}

func (lw *logrWrapper) Enabled() bool {
	return lw.level <= int(logrus.GetLevel())
}

func (lw *logrWrapper) Error(err error, msg string, keysAndValues ...interface{}) {
	lw.withKeysAndValues(keysAndValues).Error(msg, " ", err)
}

func (lw *logrWrapper) withKeysAndValues(keysAndValues []interface{}) *logrus.Entry {
	fields := logrus.Fields{}
	for i := 0; i < len(keysAndValues); i += 2 {
		var v interface{}
		if i+1 < len(keysAndValues) {
			v = keysAndValues[i+1]
		}
		fields[fmt.Sprint(keysAndValues[i])] = v
	}

	return lw.entry.WithFields(fields)
}

func (lw *logrWrapper) Info(msg string, keysAndValues ...interface{}) {
	if !lw.Enabled() {
		return
	}
	lw.withKeysAndValues(keysAndValues).Info(msg)
}

func (lw *logrWrapper) V(level int) logr.InfoLogger {
	return &logrWrapper{
		entry: lw.entry,
		level: level,
	}
}

func (lw *logrWrapper) WithValues(keysAndValues ...interface{}) logr.Logger {
	return &logrWrapper{
		entry: lw.withKeysAndValues(keysAndValues),
		level: lw.level,
	}
}

func (lw *logrWrapper) WithName(name string) logr.Logger {
	return &logrWrapper{
		entry: lw.withKeysAndValues([]interface{}{name, ""}),
		level: lw.level,
	}
}

func LogrWrapper(logger *logrus.Entry) logr.Logger {
	return &logrWrapper{
		entry: logger,
		level: int(logrus.GetLevel()),
	}
}
