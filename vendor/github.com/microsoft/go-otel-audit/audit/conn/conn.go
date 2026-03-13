// Package conn provides interfaces for audit server connections.
package conn

import (
	"context"

	"github.com/microsoft/go-otel-audit/audit/msgs"
)

//go:generate stringer -type=Type -linecomment

// Type represents the type of the audit connection.
type Type uint8

const (
	TypeUnknown      Type = 0 // Unknown
	TypeNoOP         Type = 1 // NoOP
	TypeDomainSocket Type = 2 // UnixDomainSocket
	TypeTCP          Type = 3 // TCP
)

// Audit represents a connection to a remote audit server.
// This is internal only and implemenation by external pacakges is not supported.
type Audit interface {
	// Type returns the type of the audit connection.
	Type() Type
	// Write writes a message to the remote audit server.
	Write(context.Context, msgs.Msg) error
	// CloseSend closes the send channel to the remote audit server.
	CloseSend(context.Context) error

	private() // prevent external implementations of the interface.
}
