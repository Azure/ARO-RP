package recover

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

// Panic recovers a panic
func Panic(log *logrus.Entry) {
	if e := recover(); e != nil {
		if log != nil {
			log.Error(e)
			log.Info(string(debug.Stack()))
		} else {
			fmt.Fprintln(os.Stderr, e)
			debug.PrintStack()
		}
	}
}
