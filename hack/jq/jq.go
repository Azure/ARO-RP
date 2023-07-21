// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

package main

import (
	"os"

	"github.com/itchyny/gojq/cli"
)

func main() {
	os.Exit(cli.Run())
}
