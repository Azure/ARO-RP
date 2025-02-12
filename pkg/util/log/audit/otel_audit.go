package audit

import (
	"context"
	"flag"
	"strings"

	"github.com/microsoft/go-otel-audit/audit"
	"github.com/microsoft/go-otel-audit/audit/base"
	"github.com/microsoft/go-otel-audit/audit/conn"
	"github.com/microsoft/go-otel-audit/audit/msgs"
)

const (
	auditLogQueueSize = 4000
	Unknown           = "Unknown"
)

type Client interface {
	Send(ctx context.Context, msg msgs.Msg) error
}

type Audit struct {
	Client *audit.Client
}

var _ Client = (*Audit)(nil)

func NewOtelAuditClient() (Client, error) {

	if isTestEnv() {
		return initializeNoOpOtelAuditClient()
	}

	return initializeOtelAuditClient()
}

// https://eng.ms/docs/products/geneva/collect/instrument/opentelemetryaudit/golang/linux/installation
func initializeOtelAuditClient() (Client, error) {
	newConn := func() (conn.Audit, error) {
		return conn.NewDomainSocket()
	}

	client, err := audit.New(newConn, audit.WithAuditOptions(base.WithSettings(base.Settings{QueueSize: auditLogQueueSize})))
	if err != nil {
		return nil, err
	}

	return &Audit{
		Client: client,
	}, nil
}

func (a *Audit) Send(ctx context.Context, msg msgs.Msg) error {
	return a.Client.Send(ctx, msg)
}

type MockAudit struct {
	Client *audit.Client
}

var _ Client = (*MockAudit)(nil)

// initializeNoOpOtelAuditClient creates a new no-op audit client.
// NoOP is a no-op connection to the remote audit server used during testing.
func initializeNoOpOtelAuditClient() (Client, error) {
	newNoOpConn := func() (conn.Audit, error) {
		return conn.NewNoOP(), nil
	}

	client, err := audit.New(newNoOpConn)
	if err != nil {
		return nil, err
	}

	return &Audit{
		Client: client,
	}, nil
}

func (a *MockAudit) Send(ctx context.Context, msg msgs.Msg) error {
	return nil
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

// Validate ensures that all required fields in the Record are set to default values if they are empty or invalid.
// It modifies the Record in place to ensure it meets the expected structure and data requirements.
func Validate(r *msgs.Record) {
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

func isTestEnv() bool {
	return flag.Lookup("test.v") != nil
}
