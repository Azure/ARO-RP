package audit

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/microsoft/go-otel-audit/audit"
	"github.com/microsoft/go-otel-audit/audit/conn"
	"github.com/microsoft/go-otel-audit/audit/msgs"
)

type Audit struct {
	Client *audit.Client
}

// New creates a new audit client based on the connection type (uds or tcp)
func New(connectionType string) *Audit {
	audit := &Audit{}

	if strings.ToLower(connectionType) == "uds" {
		audit.newUDSCon()
	} else {
		audit.newTCPCon("localhost:8080")
	}

	return audit
}

func (a *Audit) newUDSCon() {
	cc := func() (conn.Audit, error) {
		return conn.NewDomainSocket()
	}
	a.smartClient(cc)
}

func (a *Audit) newTCPCon(addr string) {
	cc := func() (conn.Audit, error) {
		return conn.NewTCPConn(addr)
	}
	a.smartClient(cc)
}

func (a *Audit) smartClient(cc func() (conn.Audit, error)) error {
	c, err := audit.New(cc)
	if err != nil {
		return err
	}
	a.Client = c

	return nil
}

// GetAuditRecord returns a new audit record
func GetAuditRecord() *msgs.Record {
	return &msgs.Record{}
}

// GetAuditMessage returns a new audit message based on the Msg type (dataplane or controlplane)
func GetAuditMessage(t msgs.Type) (*msgs.Msg, error) {
	msg, err := msgs.New(t)

	if err != nil {
		return nil, err
	}

	return &msg, nil
}

func (a *Audit) SendAuditMessage(ctx context.Context, msg *msgs.Msg) error {
	return a.Client.Send(ctx, *msg)
}

func GetCustomData() map[string]any {
	return map[string]any{}
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
