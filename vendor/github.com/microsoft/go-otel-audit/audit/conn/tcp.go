//go:build linux || darwin

package conn

import (
	"net"

	"github.com/microsoft/go-otel-audit/audit/conn/internal/writer"
)

// Compile check on interface implementation.
var _ Audit = TCPConn{}

// TCPConn represents a connection to a remote audit server via a TCP socket
// This implements conn.Audit interface.
type TCPConn struct {
	*writer.Conn
}

// Type returns the type of the audit connection.
func (TCPConn) Type() Type {
	return TypeTCP
}

func (TCPConn) private() {}

// NewTCPConn creates a new connection to the remote audit server. addr is the host:port of the remote audit server.
// host can be an IP address or a hostname. If host is a hostname, it will be resolved to an IP address.
func NewTCPConn(addr string) (TCPConn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return TCPConn{}, err
	}
	return TCPConn{Conn: writer.New(conn)}, nil
}
