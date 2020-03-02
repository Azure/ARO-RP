package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
)

func run() error {
	err := generator.GenerateRPTemplates()
	if err != nil {
		return err
	}

	err = generator.GenerateNSGTemplates()
	if err != nil {
		return err
	}

	err = generator.GenerateRPParameterTemplate()
	if err != nil {
		return err
	}

	return generator.GenerateDevelopmentTemplate()
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
