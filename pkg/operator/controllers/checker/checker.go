package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
)

type Checker interface {
	Check(context.Context) error
	Name() string
}

var errRequeue = errors.New("requeue")
