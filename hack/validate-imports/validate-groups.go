package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"sort"
	"strings"
)

const local = "github.com/Azure/ARO-RP"

type importType int

// at most one import group of each type may exist in a validated source file,
// specifically in the order declared below
const (
	importStd   importType = 1 << iota // go standard library
	importDot                          // "." imports (ginkgo and gomega)
	importOther                        // non-local imports
	importLocal                        // local imports
)

func typeForImport(imp *ast.ImportSpec) importType {
	path := strings.Trim(imp.Path.Value, `"`)

	switch {
	case imp.Name != nil && imp.Name.Name == ".":
		return importDot
	case strings.HasPrefix(path, local+"/"):
		return importLocal
	case strings.ContainsRune(path, '.'):
		return importOther
	default:
		return importStd
	}
}

func validateGroups(path string, fset *token.FileSet, f *ast.File) (errs []error) {
	b, err := os.ReadFile(path)
	if err != nil {
		errs = append(errs, err)
		return
	}
	// some generated files are conflicting with this rule so we exclude those
	if bytes.Contains(b, []byte("DO NOT EDIT.")) {
		return
	}
	var groups [][]*ast.ImportSpec

	for i, imp := range f.Imports {
		// if there's more than one line between this and the previous import,
		// break open a new import group
		if i == 0 || fset.Position(f.Imports[i].Pos()).Line-fset.Position(f.Imports[i-1].Pos()).Line > 1 {
			groups = append(groups, []*ast.ImportSpec{})
		}

		groups[len(groups)-1] = append(groups[len(groups)-1], imp)
	}

	// seenTypes holds a bitmask of the importTypes seen up to this point, so
	// that we can detect duplicate groups.  We can also detect misordered
	// groups, because when we set a bit (say 0b0100), we actually set all the
	// trailing bits (0b0111) as sentinels
	var seenTypes importType

	for groupnum, group := range groups {
		if !sort.SliceIsSorted(group, func(i, j int) bool { return group[i].Path.Value < group[j].Path.Value }) {
			errs = append(errs, fmt.Errorf("group %d: imports are not sorted", groupnum+1))
		}

		groupImportType := typeForImport(group[0])
		if (seenTypes & groupImportType) != 0 { // check if single bit is already set...
			errs = append(errs, fmt.Errorf("group %d: duplicate group or invalid group ordering", groupnum+1))
		}
		seenTypes |= groupImportType<<1 - 1 // ...but set all trailing bits

		for _, imp := range group {
			if typeForImport(imp) != groupImportType {
				errs = append(errs, fmt.Errorf("group %d: mixed import type", groupnum+1))
				break
			}
		}
	}

	return
}
