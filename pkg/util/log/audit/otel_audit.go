package audit

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/microsoft/go-otel-audit/audit"
	"github.com/microsoft/go-otel-audit/audit/base"
	"github.com/microsoft/go-otel-audit/audit/conn"
	"github.com/microsoft/go-otel-audit/audit/msgs"
	"github.com/sirupsen/logrus"
)

const (
	Unknown = "Unknown"
)

type Client interface {
	Send(ctx context.Context, msg msgs.Msg, options ...base.SendOption) error
}

func NewOtelAuditClient(auditLogQueueSize int, isDevEnv bool) (Client, error) {
	if isDevEnv {
		return initializeNoOpOtelAuditClient()
	}

	return initializeOtelAuditClient(auditLogQueueSize)
}

// https://eng.ms/docs/products/geneva/collect/instrument/opentelemetryaudit/golang/linux/installation
func initializeOtelAuditClient(auditLogQueueSize int) (Client, error) {
	return audit.New(
		func() (conn.Audit, error) {
			return conn.NewDomainSocket()
		},
		audit.WithAuditOptions(
			base.WithSettings(
				base.Settings{
					QueueSize: auditLogQueueSize,
				},
			),
		),
	)
}

// initializeNoOpOtelAuditClient creates a new no-op audit client.
// NoOP is a no-op connection to the remote audit server used during E2E testing or development environment.
func initializeNoOpOtelAuditClient() (Client, error) {
	return audit.New(
		func() (conn.Audit, error) {
			return conn.NewNoOP(), nil
		},
	)
}

func GetOperationType(method string) msgs.OperationType {
	switch method {
	case "GET":
		return msgs.Read
	case "POST":
		return msgs.Create
	case "PUT":
		return msgs.Update
	case "DELETE":
		return msgs.Delete
	default:
		return msgs.UnknownOperationType
	}
}

func CreateOtelAuditMsg(log *logrus.Entry, r *http.Request) msgs.Msg {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Errorf("failed to split host and port for remote request addr %q: %s", r.RemoteAddr, err)
	}

	addr, err := msgs.ParseAddr(host)
	if err != nil {
		log.Errorf("failed to parse address for host %q: %s", host, err)
	}

	record := msgs.Record{
		CallerIpAddress:              addr,
		OperationCategories:          []msgs.OperationCategory{msgs.ResourceManagement},
		OperationCategoryDescription: "Client Resource Management via frontend",
		OperationAccessLevel:         "Azure RedHat OpenShift Contributor Role",
		OperationName:                fmt.Sprintf("%s %s", r.Method, r.URL.Path),
		CallerAgent:                  r.UserAgent(),
		OperationType:                GetOperationType(r.Method),
		OperationResult:              msgs.Success,
	}

	msg := msgs.Msg{
		Type:   msgs.ControlPlane,
		Record: record,
	}

	return msg
}

// EnsureDefaults ensures that all required fields in the Record are set to default values if they are empty or invalid.
// It modifies the Record in place to ensure it meets the expected structure and data requirements.
func EnsureDefaults(r *msgs.Record) {
	setDefault := func(value *string, defaultValue string) {
		if *value == "" {
			*value = defaultValue
		}
	}

	setDefault(&r.OperationName, Unknown)
	setDefault(&r.OperationAccessLevel, Unknown)
	setDefault(&r.CallerAgent, Unknown)

	if len(r.OperationCategories) == 0 {
		r.OperationCategories = []msgs.OperationCategory{msgs.ResourceManagement}
	}

	for _, category := range r.OperationCategories {
		if category == msgs.OCOther && r.OperationCategoryDescription == "" {
			r.OperationCategoryDescription = "Other"
		}
	}

	if r.OperationResult == msgs.Failure && r.OperationResultDescription == "" {
		r.OperationResultDescription = Unknown
	}

	if len(r.CallerIdentities) == 0 {
		r.CallerIdentities = map[msgs.CallerIdentityType][]msgs.CallerIdentityEntry{
			msgs.ApplicationID: {
				{Identity: Unknown, Description: Unknown},
			},
		}
	}

	for identityType, identities := range r.CallerIdentities {
		if len(identities) == 0 {
			r.CallerIdentities[identityType] = []msgs.CallerIdentityEntry{{Identity: Unknown, Description: Unknown}}
		}
	}

	if !r.CallerIpAddress.IsValid() || r.CallerIpAddress.IsUnspecified() || r.CallerIpAddress.IsLoopback() || r.CallerIpAddress.IsMulticast() {
		r.CallerIpAddress, _ = msgs.ParseAddr("192.168.1.1")
	}

	if len(r.CallerAccessLevels) == 0 {
		r.CallerAccessLevels = []string{Unknown}
	}

	for i, k := range r.CallerAccessLevels {
		if strings.TrimSpace(k) == "" {
			r.CallerAccessLevels[i] = Unknown
		}
	}

	if len(r.TargetResources) == 0 {
		r.TargetResources = map[string][]msgs.TargetResourceEntry{
			Unknown: {
				{Name: Unknown, Region: Unknown},
			},
		}
	}

	for resourceType, resources := range r.TargetResources {
		if strings.TrimSpace(resourceType) == "" {
			r.TargetResources[Unknown] = resources
			delete(r.TargetResources, resourceType)
		}

		for _, resource := range resources {
			if err := resource.Validate(); err != nil {
				resource.Name = Unknown
			}
		}
	}
}
