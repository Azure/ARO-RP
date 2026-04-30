// Package writer implements a writer to a remote audit server of any type that can be connected to
// by a net.Conn object.
package writer

import (
	"bufio"
	"context"
	"net"
	"time"

	"github.com/microsoft/go-otel-audit/audit/msgs"
)

type now func() time.Time

// Conn is a generic writer to a remote audit server. It can use anything
// that implements the net.Conn interface.
type Conn struct {
	conn net.Conn
	buff *bufio.Writer

	now now
}

// New creates a new connection to the remote audit server.
func New(conn net.Conn) *Conn {
	return &Conn{conn: conn, buff: bufio.NewWriter(conn), now: time.Now}
}

// Write writes a message to the remote audit server. Setting a timeout
// on the context will set the write deadline.
func (c *Conn) Write(ctx context.Context, msg msgs.Msg) error {
	deadline := c.now().Add(15 * time.Second)
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	}
	c.conn.SetWriteDeadline(deadline)

	var b []byte
	var err error
	b, err = msgs.MarshalMsgpack(msg)
	if err != nil {
		return err
	}

	_, err = c.buff.Write(b)
	return err
}

// CloseSend closes the send channel to the remote audit server.
func (c *Conn) CloseSend(context.Context) error {
	if err := c.buff.Flush(); err != nil {
		return err
	}

	return c.conn.Close()
}
