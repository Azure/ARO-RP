package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
)

type logrWrapper struct {
	entry *logrus.Entry
}

func (lw *logrWrapper) Init(info logr.RuntimeInfo) {
}

func (lw *logrWrapper) Enabled(level int) bool {
	return level <= int(logrus.GetLevel())
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

func (lw *logrWrapper) Info(level int, msg string, keysAndValues ...interface{}) {
	lw.withKeysAndValues(keysAndValues).Info(msg)
}

func (lw *logrWrapper) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return &logrWrapper{
		entry: lw.withKeysAndValues(keysAndValues),
	}
}

func (lw *logrWrapper) WithName(name string) logr.LogSink {
	return &logrWrapper{
		entry: lw.withKeysAndValues([]interface{}{name, ""}),
	}
}

func LogrWrapper(logger *logrus.Entry) logr.Logger {
	return logr.New(&logrWrapper{entry: logger})
}

type logrHook struct{}

func (logrHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is an ugly hack to attempt to correct the caller information when
// logrWrapper is in use.  If the entry Caller refers to a function in this
// package, it re-fetches the backtrace again, attempts to find the frame
// matching Caller and replace it with its parent.
func (logrHook) Fire(log *logrus.Entry) error {
	if log.Caller == nil || !strings.HasPrefix(log.Caller.File, pkgpath+"/") {
		return nil
	}

	pc := make([]uintptr, 10)
	count := runtime.Callers(1, pc)
	frames := runtime.CallersFrames(pc[:count])

	for {
		frame, more := frames.Next()

		if frame == *log.Caller && more {
			frame, _ = frames.Next()
			log.Caller = &frame
			break
		}

		if !more {
			break
		}
	}

	return nil
}
