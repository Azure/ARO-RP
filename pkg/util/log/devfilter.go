package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"slices"

	"github.com/sirupsen/logrus"
)

// DevFilterFormatter logs minimal output to make viewing logs in a local
// development environment easier.
type DevFilterFormatter struct {
	innerFormatter logrus.Formatter
}

func (f *DevFilterFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	modifiedEntry := entry.Dup()
	modifiedEntry.Data = make(logrus.Fields)
	modifiedEntry.Time = entry.Time
	modifiedEntry.Message = entry.Message
	modifiedEntry.Level = entry.Level
	modifiedEntry.Caller = entry.Caller
	modifiedEntry.Context = entry.Context

	allowedFields := []string{
		"resource_id", "version", "component", "cluster_deployment_namespace",
		// azure roundtrip logging
		"LOGKIND", "request_URL", "response_status_code",
	}

	for k, v := range entry.Data {
		if slices.Contains(allowedFields, k) {
			modifiedEntry.Data[k] = v
		}
	}

	return f.innerFormatter.Format(modifiedEntry)
}

func NewDevFilterFormatter() logrus.Formatter {
	return &DevFilterFormatter{
		innerFormatter: &logrus.TextFormatter{
			FullTimestamp:    true,
			CallerPrettyfier: relativeFilePathPrettier,
		},
	}
}
