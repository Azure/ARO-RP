package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/deploy"
)

func run() error {
	err := deploy.GenerateRPTemplates()
	if err != nil {
		return err
	}

	err = deploy.GenerateNSGTemplates()
	if err != nil {
		return err
	}

	return deploy.GenerateRPParameterTemplate()
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
