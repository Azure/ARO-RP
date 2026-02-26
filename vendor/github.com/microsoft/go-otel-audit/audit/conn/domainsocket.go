//go:build linux || darwin

package conn

import (
	"fmt"
	"net"
	"os"

	"github.com/microsoft/go-otel-audit/audit/conn/internal/writer"
)

// Compile check on interface implementation.
var _ Audit = DomainSocketConn{}

// DomainSocketConn represents a connection to a remote audit server via a unix domain socket
// set to SOCK_STREAM. This implements conn.Audit interface.
type DomainSocketConn struct {
	*writer.Conn
}

// Type returns the type of the audit connection.
func (DomainSocketConn) Type() Type {
	return TypeDomainSocket
}

func (DomainSocketConn) private() {}

type dsOptions struct {
	path string
}

func (o *dsOptions) setDefaults() {
	// https://eng.ms/docs/products/geneva/collect/instrument/linux/fluent
	// Each path works the same, but they upload to different tenants.
	const (
		// asaSocket is the socket that is configured if AzSecPack AutoConfig is enabled.
		asaSocket = "/var/run/mdsd/asa/default_fluent.socket"
		// fluentSocket is the socket that is configured if AzSecPack AutoConfig is disabled
		// or out of scope.
		fluentSocket = "/var/run/mdsd/default_fluent.socket"
	)
	if o.path != "" {
		return
	}

	o.path = asaSocket
	_, err := os.Stat(o.path)
	if err != nil {
		o.path = fluentSocket
	}
}

// DSOption is an option for NewDomainSocket.
type DSOption func(*dsOptions) error

// DSPath sets the path to the unix domain socket. This is usually automatically detected and should
// only be overridden in very specific circumstances.
func DSPath(path string) DSOption {
	return func(o *dsOptions) error {
		o.path = path
		return nil
	}
}

// NewDomainSocket creates a new connection to the remote audit server.
func NewDomainSocket(options ...DSOption) (DomainSocketConn, error) {
	opts := dsOptions{}

	for _, o := range options {
		if err := o(&opts); err != nil {
			return DomainSocketConn{}, err
		}
	}
	opts.setDefaults()

	conn, err := net.Dial("unix", opts.path)
	if err != nil {
		return DomainSocketConn{}, fmt.Errorf("failed to connect to audit server(%s): %w", opts.path, err)
	}
	return DomainSocketConn{Conn: writer.New(conn)}, nil
}
