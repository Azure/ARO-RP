package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"net"
)

// recorder is a net.Conn wrapper.  When it is created, record is true.  The
// recorder copies all bytes read into a buffer so that they can be replayed
// later.  If any bytes are written to the connection, they are dropped on the
// floor.  Later, when record set to false, Read() will replay all of the bytes
// from the buffer, then continue reading; Write() will write bytes as usual.
type recorder struct {
	net.Conn
	buf    *bytes.Buffer
	record bool
}

func newRecorder(c net.Conn) *recorder {
	return &recorder{
		Conn:   c,
		buf:    &bytes.Buffer{},
		record: true,
	}
}

func (r *recorder) Read(b []byte) (int, error) {
	if r.record {
		n, err := r.Conn.Read(b)
		r.buf.Write(b[:n])
		return n, err
	}

	if r.buf.Len() > 0 {
		return r.buf.Read(b)
	}

	return r.Conn.Read(b)
}

func (r *recorder) Write(b []byte) (int, error) {
	if r.record {
		return len(b), nil
	}

	return r.Conn.Write(b)
}
