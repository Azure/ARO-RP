package template

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/util/arm"
)

var apiVersions = map[string]string{
	"authorization": "2015-07-01",
	"compute":       "2019-03-01",
	"network":       "2019-07-01",
	"privatedns":    "2018-09-01",
	"storage":       "2019-04-01",
}

type Template interface {
	Deploy(ctx context.Context) error
	Generate() *arm.Template
}
