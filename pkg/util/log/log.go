package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"flag"
	"fmt"
	"runtime"
	"strings"

	"github.com/coreos/go-systemd/journal"
	"github.com/sirupsen/logrus"
)

var (
	_, thisfile, _, _ = runtime.Caller(0)
	repopath          = strings.Replace(thisfile, "pkg/util/log/log.go", "", -1)

	loglevel = flag.String("loglevel", "info", "{panic,fatal,error,warning,info,debug,trace}")
)

// GetLogger returns a consistently configured log entry
func GetLogger() *logrus.Entry {
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		CallerPrettyfier: relativeFilePathPrettier,
	})

	if journal.Enabled() {
		logrus.AddHook(&journaldHook{})
	}

	log := logrus.NewEntry(logrus.StandardLogger())

	l, err := logrus.ParseLevel(*loglevel)
	if err == nil {
		logrus.SetLevel(l)
	} else {
		log.Warn(err)
	}

	return log
}

func relativeFilePathPrettier(f *runtime.Frame) (string, string) {
	file := strings.TrimPrefix(f.File, repopath)
	function := f.Function[strings.LastIndexByte(f.Function, '/')+1:]
	return fmt.Sprintf("%s()", function), fmt.Sprintf(" %s:%d", file, f.Line)
}
