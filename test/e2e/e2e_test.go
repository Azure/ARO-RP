//+build e2e

package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"flag"
	"math/rand"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/format"
	"github.com/sirupsen/logrus"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

var (
	gitCommit = "unknown"
)

func TestE2E(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	flag.Parse()
	logrus.SetOutput(GinkgoWriter)
	Log = utillog.GetLogger()
	Log.Infof("e2e tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	format.TruncatedDiff = false
	RunSpecs(t, "e2e tests")
}
