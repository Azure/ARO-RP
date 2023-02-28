package listener

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/Azure/ARO-RP/test/util/bufferedpipe"
)

type addr struct{}

func (addr) Network() string { return "testlistener" }
func (addr) String() string  { return "testlistener" }

type Listener struct {
	c    chan net.Conn
	once sync.Once
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
	l.once.Do(func() {
		close(l.c)
	})

	return nil
}

func (*Listener) Addr() net.Addr {
	return &addr{}
}

func (l *Listener) DialContext(context.Context, string, string) (net.Conn, error) {
	c1, c2 := bufferedpipe.New()
	l.c <- c1
	return c2, nil
}

func (l *Listener) Enqueue(c net.Conn) {
	l.c <- c
}
