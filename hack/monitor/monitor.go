package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"io"
	"net"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func run(log *logrus.Entry) error {
	os.Remove("mdm_statsd.socket")

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	l, err := net.Listen("unix", "mdm_statsd.socket")
	if err != nil {
		return err
	}

	log.Print("listening")

	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}

			go func() {
				_, err := io.Copy(os.Stdout, c)
				if err != nil {
					log.Error(err)
				}
			}()
		}
	}()

	<-sigint

	return l.Close()
}

func main() {
	log := utillog.GetLogger()

	err := run(log)
	if err != nil {
		panic(err)
	}
}
