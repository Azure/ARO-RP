package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	_ "embed"
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

func isStandardLibrary(path string) bool {
	return !strings.ContainsRune(strings.SplitN(path, "/", 2)[0], '.')
}

func validateUnderscoreImport(path string) error {
	if regexp.MustCompile(`^github\.com/Azure/ARO-RP/pkg/api/(admin|v[^/]+)$`).MatchString(path) {
		return nil
	}

	switch path {
	case "net/http/pprof",
		"github.com/Azure/ARO-RP/pkg/util/scheme",
		"embed":
		return nil
	}

	return fmt.Errorf("invalid _ import %s", path)
}

func validateImports(path string, fset *token.FileSet, f *ast.File) []error {
	for _, prefix := range []string{
		"pkg/client/",
		"pkg/database/cosmosdb/zz_generated_",
		"pkg/operator/apis",
		"pkg/operator/clientset",
		"pkg/operator/mocks/",
		"pkg/util/mocks/",
	} {
		if strings.HasPrefix(path, prefix) {
			return nil
		}
	}

	errs := make([]error, 0)
	for _, imp := range f.Imports {
		if err := validateImport(imp); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func validateImport(imp *ast.ImportSpec) error {
	packageName := strings.Trim(imp.Path.Value, `"`)

	if imp.Name != nil && imp.Name.Name == "." {
		//accept dotimports because we check them with golangci-lint
		return nil
	}

	if imp.Name != nil && imp.Name.Name == "_" {
		err := validateUnderscoreImport(packageName)
		if err != nil {
			return err
		}
		return nil
	}

	switch packageName {
	case "sigs.k8s.io/yaml", "gopkg.in/yaml.v2":
		return fmt.Errorf("%s is imported; use github.com/ghodss/yaml", packageName)
	case "github.com/google/uuid", "github.com/satori/go.uuid":
		return fmt.Errorf("%s is imported; use github.com/gofrs/uuid", packageName)
	}

	if strings.HasPrefix(packageName, "github.com/Azure/azure-sdk-for-go/profiles") {
		return fmt.Errorf("%s is imported; use github.com/Azure/azure-sdk-for-go/services/*", packageName)
	}

	if strings.HasSuffix(packageName, "/scheme") &&
		packageName != "k8s.io/client-go/kubernetes/scheme" {
		return fmt.Errorf("%s is imported; should probably use k8s.io/client-go/kubernetes/scheme", packageName)
	}

	if isStandardLibrary(packageName) {
		if imp.Name != nil {
			return fmt.Errorf("overridden import %s", packageName)
		}
		return nil
	}

	return nil
}
