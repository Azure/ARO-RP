package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
)

func run() error {
	// dev artifacts
	err := generator.New(false).Artifacts()
	if err != nil {
		return err
	}

	// prod artifacts
	return generator.New(true).Artifacts()
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
