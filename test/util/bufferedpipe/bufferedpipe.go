package bufferedpipe

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

// New returns two net.Conns representing either side of a buffered pipe.  It's
// like net.Pipe() but with buffering.  Note that deadlines are currently not
// implemented.
func New() (net.Conn, net.Conn) {
	p := &p{
		cond: sync.NewCond(&sync.Mutex{}),
	}

	return &conn{p, 0}, &conn{p, 1}
}

type p struct {
	cond   *sync.Cond
	buf    [2]bytes.Buffer
	closed [2]bool
}

type conn struct {
	p *p
	n int
}

func (c *conn) Read(b []byte) (int, error) {
	c.p.cond.L.Lock()
	defer c.p.cond.L.Unlock()

	for {
		if c.p.closed[c.n] {
			// Read() concurrently with, or after Close()
			return 0, errors.New("connection closed")
		}

		if c.p.buf[c.n^1].Len() > 0 {
			return c.p.buf[c.n^1].Read(b)
		}

		if c.p.closed[c.n^1] {
			// Other side closed and read buffer is drained
			return 0, io.EOF
		}

		c.p.cond.Wait()
	}
}

func (c *conn) Write(b []byte) (int, error) {
	c.p.cond.L.Lock()
	defer c.p.cond.L.Unlock()

	if c.p.closed[c.n] {
		return 0, errors.New("connection closed")
	}

	c.p.cond.Broadcast()
	return c.p.buf[c.n].Write(b)
}

func (c *conn) Close() error {
	c.p.cond.L.Lock()
	defer c.p.cond.L.Unlock()

	if c.p.closed[c.n] {
		return errors.New("connection closed")
	}

	c.p.closed[c.n] = true
	c.p.cond.Broadcast()
	return nil
}

func (c *conn) LocalAddr() net.Addr              { return &addr{} }
func (c *conn) RemoteAddr() net.Addr             { return &addr{} }
func (c *conn) SetDeadline(time.Time) error      { return errors.New("not implemented") }
func (c *conn) SetReadDeadline(time.Time) error  { return errors.New("not implemented") }
func (c *conn) SetWriteDeadline(time.Time) error { return errors.New("not implemented") }

type addr struct{}

func (addr) Network() string { return "bufferedpipe" }
func (addr) String() string  { return "bufferedpipe" }
