package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	_ "embed"
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
)

//go:embed allowed-import-names.yaml
var allowedNamesYaml []byte

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

type importValidator struct {
	AllowedNames map[string][]string `json:"allowedImportNames"`
}

func initValidator() importValidator {
	allowed := importValidator{}
	err := yaml.Unmarshal(allowedNamesYaml, &allowed)
	if err != nil {
		log.Fatalf("error while unmarshalling allowed import names. err: %s", err)
	}
	return allowed
}

func (validator importValidator) isOkFromYaml(name, importedAs string) (bool, []string) {
	for _, v := range validator.AllowedNames[name] {
		if importedAs == v {
			return true, nil
		}
	}
	return false, validator.AllowedNames[name]
}

func (validator importValidator) validateImportName(name, importedAs string) error {
	isAllowed, names := validator.isOkFromYaml(name, importedAs)

	if isAllowed {
		return nil
	}

	isAllowedFromRegex, namesRegex := isOkFromRegex(name, importedAs)

	if isAllowedFromRegex {
		return nil
	}

	names = append(names, namesRegex...)

	return fmt.Errorf("%s is imported as %q, should be %q", name, importedAs, names)
}

func isOkFromRegex(name, importedAs string) (bool, []string) {
	acceptableNames := acceptableNamesRegex(name)

	for _, v := range acceptableNames {
		if v == importedAs {
			return true, nil
		}
	}
	return false, acceptableNames
}

// acceptableNamesRegex returns a list of acceptable names for an import; empty
// string = no import override; nil list = don't care
func acceptableNamesRegex(path string) []string {
	m := regexp.MustCompile(`^github\.com/Azure/ARO-RP/pkg/api/(v[^/]*[0-9])$`).FindStringSubmatch(path)
	if m != nil {
		return []string{m[1]}
	}

	m = regexp.MustCompile(`^github\.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/([^/]+)/redhatopenshift$`).FindStringSubmatch(path)
	if m != nil {
		return []string{"mgmtredhatopenshift" + strings.ReplaceAll(m[1], "-", "")}
	}

	m = regexp.MustCompile(`^github\.com/Azure/ARO-RP/pkg/(dbtoken|deploy|gateway|mirror|monitor|operator|portal)$`).FindStringSubmatch(path)
	if m != nil {
		return []string{"", "pkg" + m[1]}
	}

	m = regexp.MustCompile(`^github\.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/([^/]+)/redhatopenshift$`).FindStringSubmatch(path)
	if m != nil {
		return []string{"redhatopenshift" + strings.ReplaceAll(m[1], "-", "")}
	}

	m = regexp.MustCompile(`^github\.com/Azure/ARO-RP/pkg/util/(log|net|pem|tls)$`).FindStringSubmatch(path)
	if m != nil {
		return []string{"util" + m[1]}
	}

	m = regexp.MustCompile(`^github\.com/Azure/ARO-RP/pkg/util/mocks/(?:.+/)?([^/]+)$`).FindStringSubmatch(path)
	if m != nil {
		return []string{"mock_" + m[1]}
	}

	m = regexp.MustCompile(`^github\.com/Azure/ARO-RP/pkg/operator/mocks/(?:.+/)?([^/]+)$`).FindStringSubmatch(path)
	if m != nil {
		return []string{"mock_" + m[1]}
	}

	m = regexp.MustCompile(`^github\.com/Azure/azure-sdk-for-go/services/(?:preview/)?(?:[^/]+)/mgmt/(?:[^/]+)/([^/]+)$`).FindStringSubmatch(path)
	if m != nil {
		return []string{"mgmt" + m[1]}
	}

	m = regexp.MustCompile(`^github\.com/openshift/api/([^/]+)/(v[^/]+)$`).FindStringSubmatch(path)
	if m != nil {
		return []string{m[1] + m[2]}
	}

	m = regexp.MustCompile(`^github\.com/openshift/client-go/([^/]+)/clientset/versioned$`).FindStringSubmatch(path)
	if m != nil {
		return []string{m[1] + "client"}
	}

	m = regexp.MustCompile(`^github\.com/openshift/client-go/([^/]+)/clientset/versioned/fake$`).FindStringSubmatch(path)
	if m != nil {
		return []string{m[1] + "fake"}
	}

	m = regexp.MustCompile(`^k8s\.io/api/([^/]+)/(v[^/]+)$`).FindStringSubmatch(path)
	if m != nil {
		return []string{m[1] + m[2]}
	}

	m = regexp.MustCompile(`^k8s\.io/kubernetes/pkg/apis/[^/]+/v[^/]+$`).FindStringSubmatch(path)
	if m != nil {
		return nil
	}

	m = regexp.MustCompile(`^k8s\.io/client-go/kubernetes/typed/([^/]+)/(v[^/]+)$`).FindStringSubmatch(path)
	if m != nil {
		return []string{m[1] + m[2] + "client"}
	}

	return []string{""}
}

func importedAs(spec *ast.ImportSpec) string {
	if spec == nil {
		return ""
	}

	if spec.Name == nil {
		return ""
	}

	return spec.Name.Name
}

func validateImports(path string, fset *token.FileSet, f *ast.File) []error {
	for _, prefix := range []string{
		"pkg/client/",
		"pkg/hive/clientset",
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
	validator := initValidator()
	for _, imp := range f.Imports {
		if err := validator.validateImport(imp); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (validator importValidator) validateImport(imp *ast.ImportSpec) error {
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

	names := acceptableNamesRegex(packageName)

	if names == nil {
		return nil
	}

	importedAs := importedAs(imp)
	err := validator.validateImportName(packageName, importedAs)
	if err != nil {
		return err
	}

	return nil
}
