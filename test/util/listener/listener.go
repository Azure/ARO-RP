package listener

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
)

type testAddr struct{}

func (testAddr) Network() string { return "" }
func (testAddr) String() string  { return "" }

type Listener struct {
	c      chan net.Conn
	closed bool
}

func NewListener() *Listener {
	return &Listener{
		c: make(chan net.Conn),
	}
}

func (l *Listener) Accept() (net.Conn, error) {
	c, ok := <-l.c
	if !ok {
		return nil, fmt.Errorf("listener closed")
	}
	return c, nil
}

func (l *Listener) Close() error {
	if !l.closed {
		close(l.c)
		l.closed = true
	}
	return nil
}

func (*Listener) Addr() net.Addr {
	return testAddr{}
}

func (l *Listener) DialContext(context.Context, string, string) (net.Conn, error) {
	c1, c2 := net.Pipe()
	l.c <- c1
	return c2, nil
}
