package advisor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
)

type Advisor interface {
	Check(context.Context) error
	Name() string
}

var errRequeue = errors.New("requeue")
