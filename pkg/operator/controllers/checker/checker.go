package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type Checker interface {
	Check() error
	Name() string
}
