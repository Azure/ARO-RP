//go:build e2e
// +build e2e

package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"flag"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/format"
	"github.com/sirupsen/logrus"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func TestE2E(t *testing.T) {
	flag.Parse()
	logrus.SetOutput(GinkgoWriter)
	log = utillog.GetLogger()
	log.Infof("e2e tests starting, git commit %s\n", version.GitCommit)
	RegisterFailHandler(Fail)
	format.TruncatedDiff = false
	RunSpecs(t, "e2e tests")
}
