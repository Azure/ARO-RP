package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var rv int
	for _, path := range os.Args[1:] {
		if err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && strings.HasSuffix(path, ".go") {
				for _, err := range check(path) {
					fmt.Printf("%s: %v\n", path, err)
					rv = 1
				}
			}

			return nil
		}); err != nil {
			panic(err)
		}
	}
	os.Exit(rv)
}
