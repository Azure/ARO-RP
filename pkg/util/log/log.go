package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"flag"
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/coreos/go-systemd/journal"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var (
	_, thisfile, _, _ = runtime.Caller(0)
	repopath          = strings.Replace(thisfile, "pkg/util/log/log.go", "", -1)

	loglevel = flag.String("loglevel", "info", "{panic,fatal,error,warning,info,debug,trace}")

	rxTolerantResourceID = regexp.MustCompile(`(?i)^(?:/admin)?/subscriptions/([^/]+)(?:/resourceGroups/([^/]+)(?:/providers/([^/]+)/([^/]+)(?:/([^/]+))?)?)?`)
)

type logrusWrapper struct {
	entry *logrus.Entry
	level int
}

func (lw *logrusWrapper) Enabled() bool {
	return lw.level <= int(logrus.GetLevel())
}

func (lw *logrusWrapper) Error(err error, msg string, keysAndValues ...interface{}) {
	lw.withKeysAndValues(keysAndValues).Error(msg, err)
}

func (lw *logrusWrapper) withKeysAndValues(keysAndValues []interface{}) *logrus.Entry {
	if len(keysAndValues) == 0 {
		return lw.entry
	}
	key := ""
	fields := logrus.Fields{}
	for _, item := range keysAndValues {
		if key == "" {
			key = fmt.Sprint(item)
		} else {
			fields[key] = fmt.Sprint(item)
			key = ""
		}
	}
	if key != "" {
		// key with no value
		fields[key] = ""
	}

	return lw.entry.WithFields(fields)
}

func (lw *logrusWrapper) Info(msg string, keysAndValues ...interface{}) {
	if !lw.Enabled() {
		return
	}
	lw.withKeysAndValues(keysAndValues).Info(msg)
}

func (lw *logrusWrapper) V(level int) logr.InfoLogger {
	return &logrusWrapper{
		entry: lw.entry,
		level: level,
	}
}

func (lw *logrusWrapper) WithValues(keysAndValues ...interface{}) logr.Logger {
	return &logrusWrapper{
		entry: lw.withKeysAndValues(keysAndValues),
		level: lw.level,
	}
}

func (lw *logrusWrapper) WithName(name string) logr.Logger {
	return &logrusWrapper{
		entry: lw.withKeysAndValues([]interface{}{name, ""}),
		level: lw.level,
	}
}

func GetRLogger(logger *logrus.Entry) logr.Logger {
	return &logrusWrapper{
		entry: logger,
		level: int(logrus.GetLevel()),
	}
}

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

// EnrichWithPath parses the URL path for part or all of an Azure resource ID
// and sets log fields accordingly
func EnrichWithPath(log *logrus.Entry, path string) *logrus.Entry {
	m := rxTolerantResourceID.FindStringSubmatch(path)
	if m == nil {
		return log
	}

	fields := logrus.Fields{}
	if m[1] != "" {
		fields["subscription_id"] = m[1]
	}
	if m[2] != "" {
		fields["resource_group"] = m[2]
	}
	if m[5] != "" {
		fields["resource_name"] = m[5]
		fields["resource_id"] = "/subscriptions/" + m[1] + "/resourceGroups/" + m[2] + "/providers/" + m[3] + "/" + m[4] + "/" + m[5]
	}

	return log.WithFields(fields)
}

// EnrichWithCorrelationData sets log fields based on an optional
// correlationData struct
func EnrichWithCorrelationData(log *logrus.Entry, correlationData *api.CorrelationData) *logrus.Entry {
	if correlationData == nil {
		return log
	}

	return log.WithFields(logrus.Fields{
		"correlation_id":        correlationData.CorrelationID,
		"client_request_id":     correlationData.ClientRequestID,
		"request_id":            correlationData.RequestID,
		"client_principal_name": correlationData.ClientPrincipalName,
	})
}

// EnrichWithResourceID sets log fields based on a resource ID
func EnrichWithResourceID(log *logrus.Entry, resourceID string) *logrus.Entry {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		log.Error(err)
		return log
	}

	return log.WithFields(logrus.Fields{
		"resource_id":     strings.ToLower(resourceID),
		"subscription_id": strings.ToLower(r.SubscriptionID),
		"resource_group":  strings.ToLower(r.ResourceGroup),
		"resource_name":   strings.ToLower(r.ResourceName),
	})
}

func relativeFilePathPrettier(f *runtime.Frame) (string, string) {
	file := strings.TrimPrefix(f.File, repopath)
	function := stringutils.LastTokenByte(f.Function, '/')
	return fmt.Sprintf("%s()", function), fmt.Sprintf("%s:%d", file, f.Line)
}
