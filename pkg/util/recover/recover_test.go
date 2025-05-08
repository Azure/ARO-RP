package recover

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"

	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestPanic(t *testing.T) {
	h, log := testlog.New()

	func() {
		defer Panic(log)
		panic("random error")
	}()

	expected := []map[string]types.GomegaMatcher{
		{
			"msg":   gomega.Equal("random error"),
			"level": gomega.Equal(logrus.ErrorLevel),
		},
		{
			"msg":   gomega.MatchRegexp(`runtime/debug\.Stack`),
			"level": gomega.Equal(logrus.InfoLevel),
		},
	}

	err := testlog.AssertLoggingOutput(h, expected)
	if err != nil {
		t.Error(err)
	}
}
