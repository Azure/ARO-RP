//+build e2e

package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"flag"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/onsi/gomega/format"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

var (
	gitCommit = "unknown"
)

func TestE2E(t *testing.T) {
	flag.Parse()
	logrus.SetOutput(GinkgoWriter)
	log := utillog.GetLogger()
	log.Infof("e2e tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	format.TruncatedDiff = false
	RunSpecs(t, "e2e tests")
}
