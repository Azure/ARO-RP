package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"

	"github.com/sirupsen/logrus"
)

func LoggerForService(service ServiceName, logger *logrus.Entry) *logrus.Entry {
	return logger.WithField("service", strings.ReplaceAll(strings.ToLower(string(service)), "_", "-"))
}
