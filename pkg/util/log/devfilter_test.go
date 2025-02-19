package log

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
)

func TestDevFilter(t *testing.T) {
	out := &bytes.Buffer{}

	logr := &logrus.Logger{
		Out:       out,
		Formatter: NewDevFilterFormatter(),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}

	log := logrus.NewEntry(logr)
	log.Time = time.UnixMilli(10000).UTC()
	log = EnrichWithPath(log, "/subscriptions/subscriptionid/resourceGroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/foo")

	// send a log
	log.Info("test")

	contents, err := io.ReadAll(out)
	if err != nil {
		t.Fatal(err)
	}

	expectedLog := "time=\"1970-01-01T00:00:10Z\" level=info msg=test resource_id=/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename\n"

	for _, e := range deep.Equal(string(contents), expectedLog) {
		t.Error(e)
	}
}
