package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"io"
	"net"
	"os"
)

func run() error {
	l, err := net.Listen("unix", "mdm_statsd.socket")
	if err != nil {
		return err
	}

	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		go io.Copy(os.Stdout, c)
	}
}

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}
