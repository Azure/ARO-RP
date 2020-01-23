package recover

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestPanic(t *testing.T) {
	logger, hook := test.NewNullLogger()
	log := logrus.NewEntry(logger)

	func() {
		defer Panic(log)
		panic("random error")
	}()

	if len(hook.Entries) != 2 {
		t.Fatal(len(hook.Entries))
	}

	if hook.Entries[0].Message != "random error" {
		t.Error(hook.Entries[0].Message)
	}

	if !strings.Contains(hook.Entries[1].Message, "runtime/debug.Stack") {
		t.Error(hook.Entries[1].Message)
	}
}
