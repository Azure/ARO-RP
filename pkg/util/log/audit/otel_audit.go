package audit

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"log"
	"strings"

	otelaudit "github.com/microsoft/go-otel-audit/audit"
	"github.com/microsoft/go-otel-audit/audit/conn"
	"github.com/microsoft/go-otel-audit/audit/msgs"
)

type Audit struct {
	Client           *otelaudit.Client
	SendAuditMessage func(a *otelaudit.Client, ctx context.Context, msg *msgs.Msg) error
	Count            int
}

// New creates a new audit client based on the connection type (uds or tcp)
func New(connectionType string, isTest bool) *Audit {
	audit := &Audit{Count: 0}

	if isTest {
		audit.newNoOpCon()
		audit.SendAuditMessage = func(c *otelaudit.Client, ctx context.Context, msg *msgs.Msg) error {
			return nil
		}
	} else {
		if strings.EqualFold(connectionType, "uds") {
			audit.newUDSCon()
		} else {
			audit.newTCPCon("localhost:29230")
		}
	}

	//TODO: gnir - Rmove after testing in INT
	log.Printf("Frontend - Client %v", audit)
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

func (a *Audit) newNoOpCon() {
	cc := func() (conn.Audit, error) {
		return conn.NewNoOP(), nil
	}
	a.smartClient(cc)
}

func (a *Audit) smartClient(cc func() (conn.Audit, error)) error {
	c, err := otelaudit.New(cc)

	//TODO: gnir - Rmove after testing in INT
	log.Printf("Frontend - Smart Client %v, %v", c, err)
	if err != nil {
		return err
	}

	a.SendAuditMessage = func(c *otelaudit.Client, ctx context.Context, msg *msgs.Msg) error { return c.Send(ctx, *msg) }

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
