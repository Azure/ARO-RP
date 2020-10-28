package checker

import "context"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type Checker interface {
	Check(context.Context) error
	Name() string
}
