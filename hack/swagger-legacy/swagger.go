package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"

	"github.com/Azure/ARO-RP/pkg/swagger"
)

func main() {
	if err := swagger.Run(os.Args[1], os.Args[2]); err != nil {
		panic(err)
	}
}
