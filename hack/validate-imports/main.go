package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var rv int
	for _, path := range os.Args[1:] {
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}

			fset := &token.FileSet{}

			f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}

			for _, err := range validateGroups(path, fset, f) {
				fmt.Printf("%s: %v\n", path, err)
				rv = 1
			}

			return nil
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			rv = 1
		}
	}
	os.Exit(rv)
}
