package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/microsoft/go-otel-audit/audit/base"
	"github.com/microsoft/go-otel-audit/audit/msgs"

	"github.com/Azure/ARO-RP/pkg/util/log/audit"
)

type MockAudit struct{}

func NewOtelAuditClient() audit.Client {
	return &MockAudit{}
}

func (a *MockAudit) Send(ctx context.Context, msg msgs.Msg, options ...base.SendOption) error {
	return nil
}
