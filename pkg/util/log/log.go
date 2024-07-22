package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"flag"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/coreos/go-systemd/v22/journal"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var (
	_, thisfile, _, _ = runtime.Caller(0)
	pkgpath           = filepath.Dir(thisfile)
	repopath          = strings.Replace(thisfile, "pkg/util/log/log.go", "", -1)

	loglevel = flag.String("loglevel", "info", "{panic,fatal,error,warning,info,debug,trace}")

	// matches URLs that look like /subscriptions/%s/providers/%s/%s
	RXProviderResourceKind = regexp.MustCompile(`^/subscriptions/([^/]+)/providers/([^/]+)/([^/]+)$`)

	// matches URLs that look like /admin/providers/%s/%s
	RXAdminProvider = regexp.MustCompile(`^/admin/providers/([^/]+)/([^/]+)$`)

	RXTolerantResourceID = regexp.MustCompile(`(?i)^(?:/admin)?/subscriptions/([^/]+)(?:/resourceGroups/([^/]+)(?:/providers/([^/]+)/([^/]+)(?:/([^/]+))?)?)?`)

	RXTolerantSubResourceID = regexp.MustCompile(`(?i)^(?:/admin)?/subscriptions/([^/]+)(?:/resourceGroups/([^/]+)(?:/providers/([^/]+)/([^/]+)/([^/]+)(?:/([^/]+))?)?)?`)
)

type ResultType string

const (
	SuccessResultType     ResultType = "Success"
	UserErrorResultType   ResultType = "UserError"
	ServerErrorResultType ResultType = "InternalServerError"
)

func MapStatusCodeToResultType(statusCode int) ResultType {
	if statusCode < 300 && statusCode >= 200 {
		return SuccessResultType
	}
	if statusCode < 500 {
		return UserErrorResultType
	}
	return ServerErrorResultType
}

func getBaseLogger() *logrus.Logger {
	logger := logrus.New()

	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	logger.AddHook(&logrHook{})

	return logger
}

// Important: Logger hooks loading order is important as they are executed in
// such order (https://github.com/sirupsen/logrus/blob/master/hooks.go#L26-L34).
//  This means we need to load all populating hooks (logrHook, auditHook)
// first, and emitting hooks (journald) last. Otherwise enriched data is lost.
// In addition to that we need to understand that in situations like this log output
// in CMD might be not the same as in journald, because we emit directly to journald.
// If to remove this, we would need additional layer for log parsing.

// GetAuditEntry returns a consistently configured audit log entry
func GetAuditEntry() *logrus.Entry {
	auditLogger := getBaseLogger()

	auditLogger.AddHook(&audit.PayloadHook{
		Payload: &audit.Payload{},
	})

	if journal.Enabled() {
		auditLogger.AddHook(&journaldHook{})
	}

	return logrus.NewEntry(auditLogger)
}

// GetLogger returns a consistently configured log entry
func GetLogger() *logrus.Entry {
	logger := getBaseLogger()

	logger.SetReportCaller(true)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		CallerPrettyfier: relativeFilePathPrettier,
	})

	if journal.Enabled() {
		logger.AddHook(&journaldHook{})
	}

	log := logrus.NewEntry(logger)

	l, err := logrus.ParseLevel(*loglevel)
	if err == nil {
		logger.SetLevel(l)
	} else {
		log.Warn(err)
	}

	return log
}

// EnrichWithPath parses the URL path for part or all of an Azure resource ID
// and sets log fields accordingly
func EnrichWithPath(log *logrus.Entry, path string) *logrus.Entry {
	m := RXTolerantResourceID.FindStringSubmatch(path)
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
		fields["resource_id"] = "/subscriptions/" + m[1] + "/resourcegroups/" + m[2] + "/providers/" + m[3] + "/" + m[4] + "/" + m[5]
	}

	return log.WithFields(fields)
}

// EnrichWithCorrelationData sets log fields based on an optional
// correlationData struct
func EnrichWithCorrelationData(log *logrus.Entry, correlationData *api.CorrelationData) *logrus.Entry {
	if log == nil {
		log = GetLogger()
	}

	if correlationData == nil {
		return log
	}

	return log.WithFields(logrus.Fields{
		"correlation_id":        correlationData.CorrelationID,
		"client_request_id":     correlationData.ClientRequestID,
		"operation_id":          correlationData.OperationID,
		"request_id":            correlationData.RequestID,
		"client_principal_name": correlationData.ClientPrincipalName,
		"request_time":          correlationData.RequestTime,
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

// EnrichWithClusterDeploymentNamespace sets log fields based on hiveProfile.Namespace
func EnrichWithClusterDeploymentNamespace(log *logrus.Entry, clusterDeploymentNamespace string) *logrus.Entry {
	if clusterDeploymentNamespace == "" {
		return log
	}

	return log.WithFields(logrus.Fields{
		"cluster_deployment_namespace": strings.ToLower(clusterDeploymentNamespace),
	})
}

// EnrichWithClusterVersion set log fields based on cluster version
func EnrichWithClusterVersion(log *logrus.Entry, version string) *logrus.Entry {
	if version == "" {
		return log
	}

	return log.WithFields(logrus.Fields{
		"version": strings.ToLower(version),
	})
}

func relativeFilePathPrettier(f *runtime.Frame) (string, string) {
	file := strings.TrimPrefix(f.File, repopath)
	function := stringutils.LastTokenByte(f.Function, '/')
	return fmt.Sprintf("%s()", function), fmt.Sprintf("%s:%d", file, f.Line)
}
