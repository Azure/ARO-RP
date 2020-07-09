package recover

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/sirupsen/logrus"

	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestPanic(t *testing.T) {

	h, log := testlog.NewCapturingLogger()

	func() {
		defer Panic(log)
		panic("random error")
	}()

	expected := []testlog.ExpectedLogEntry{
		{
			Message: "random error",
			Level:   logrus.ErrorLevel,
		},
		{
			MessageRegex: `runtime\/debug\.Stack`,
			Level:        logrus.InfoLevel,
		},
	}

	for _, e := range testlog.AssertLoggingOutput(h, expected) {
		t.Error(e)
	}
}
