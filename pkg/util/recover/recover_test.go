package recover

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/sirupsen/logrus"

	test_log "github.com/Azure/ARO-RP/test/util/log"
)

func TestPanic(t *testing.T) {

	h, log := test_log.NewCapturingLogger()

	func() {
		defer Panic(log)
		panic("random error")
	}()

	expected := []test_log.ExpectedLogEntry{
		{
			Message: "random error",
			Level:   logrus.ErrorLevel,
		},
		{
			MessageRegex: `runtime\/debug\.Stack`,
			Level:        logrus.InfoLevel,
		},
	}

	for _, e := range test_log.AssertLoggingOutput(h, expected) {
		t.Error(e)
	}
}
