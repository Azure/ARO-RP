package conn

import (
	"context"

	"github.com/microsoft/go-otel-audit/audit/msgs"
)

// Compile check on interface implementation.
var _ Audit = NoOP{}

// NoOP is a no-op connection to the remote audit server. This implements conn.Audit interface.
// Use this when you don't want to send audit messages to the remote audit server.
type NoOP struct{}

func (NoOP) Type() Type {
	return TypeNoOP
}

func (NoOP) private() {}

// NewNoOP creates a new no-op Conn that does nothing when called.
func NewNoOP() NoOP {
	return NoOP{}
}

// Write implements conn.Audit interface.
func (n NoOP) Write(context.Context, msgs.Msg) error {
	return nil
}

// CloseSend implements conn.Audit.CloseSend interface.
func (n NoOP) CloseSend(context.Context) error {
	return nil
}
