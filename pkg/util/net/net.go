package net

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"net"
	"syscall"
)

// Listen returns a listener with its send and receive buffer sizes set, such
// that sockets which are *returned* by the listener when Accept() is called
// also have those buffer sizes.
func Listen(network, address string, sz int) (net.Listener, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}

	sc, ok := l.(syscall.Conn)
	if !ok {
		return nil, errors.New("listener does not implement Syscall.Conn")
	}

	rc, err := sc.SyscallConn()
	if err != nil {
		return nil, err
	}

	err = setBuffers(rc, sz)
	if err != nil {
		return nil, err
	}

	return l, nil
}

// Dial returns a dialled connection with its send and receive buffer sizes set.
// If sz <= 0, we leave the default size.
func Dial(network, address string, sz int) (net.Conn, error) {
	return (&net.Dialer{
		Control: func(network, address string, rc syscall.RawConn) error {
			if sz <= 0 {
				return nil
			}

			return setBuffers(rc, sz)
		},
	}).Dial(network, address)
}

// read socket(7)
func setBuffers(rc syscall.RawConn, sz int) error {
	var err2 error
	err := rc.Control(func(fd uintptr) {
		err2 = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, sz)
	})
	if err2 != nil {
		return err2
	}
	if err != nil {
		return err
	}

	err = rc.Control(func(fd uintptr) {
		err2 = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, sz)
	})
	if err2 != nil {
		return err2
	}

	return err
}
