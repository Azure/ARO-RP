package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

func isStandardLibrary(path string) bool {
	return !strings.ContainsRune(strings.SplitN(path, "/", 2)[0], '.')
}

func validateDotImport(path string) error {
	switch path {
	case "github.com/onsi/ginkgo",
		"github.com/onsi/gomega":
		return nil
	}

	return fmt.Errorf("invalid . import %s", path)
}

func validateUnderscoreImport(path string) error {
	if regexp.MustCompile(`^github\.com/Azure/ARO-RP/pkg/api/(admin|v[^/]+)$`).MatchString(path) {
		return nil
	}

	switch path {
	case "net/http/pprof",
		"github.com/Azure/ARO-RP/pkg/util/scheme":
		return nil
	}

	return fmt.Errorf("invalid _ import %s", path)
}

// acceptableNames returns a list of acceptable names for an import; empty
// string = no import override; nil list = don't care
func acceptableNames(path string) []string {
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

	switch path {
	case "github.com/Azure/ARO-RP/pkg/frontend/middleware":
		return []string{"", "frontendmiddleware"}
	case "github.com/Azure/ARO-RP/pkg/metrics/statsd/cosmosdb":
		return []string{"dbmetrics"}
	case "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1":
		return []string{"arov1alpha1"}
	case "github.com/Azure/ARO-RP/pkg/operator/apis/preview.aro.openshift.io/v1alpha1":
		return []string{"aropreviewv1alpha1"}
	case "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned":
		return []string{"aroclient"}
	case "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake":
		return []string{"arofake"}
	case "github.com/Azure/ARO-RP/pkg/util/dynamichelper/discovery":
		return []string{"utildiscovery"}
	case "github.com/Azure/ARO-RP/pkg/util/namespace":
		return []string{"", "utilnamespace"}
	case "github.com/Azure/ARO-RP/pkg/util/recover":
		return []string{"", "utilrecover"}
	case "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/keyvault":
		return []string{"", "keyvaultclient"}
	case "github.com/Azure/ARO-RP/test/database":
		return []string{"testdatabase"}
	case "github.com/Azure/ARO-RP/test/util/dynamichelper":
		return []string{"testdynamichelper"}
	case "github.com/Azure/ARO-RP/test/util/log":
		return []string{"testlog"}
	case "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac":
		return []string{"azgraphrbac"}
	case "github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault":
		return []string{"azkeyvault"}
	case "github.com/Azure/azure-sdk-for-go/storage":
		return []string{"azstorage"}
	case "github.com/googleapis/gnostic/openapiv2":
		return []string{"openapi_v2"}
	case "github.com/openshift/console-operator/pkg/api":
		return []string{"consoleapi"}
	case "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1":
		return []string{"machinev1beta1"}
	case "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned":
		return []string{"maoclient"}
	case "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned/fake":
		return []string{"maofake"}
	case "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1":
		return []string{"mcv1"}
	case "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned":
		return []string{"mcoclient"}
	case "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/fake":
		return []string{"mcofake"}
	case "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/typed/machineconfiguration.openshift.io/v1":
		return []string{"mcoclientv1"}
	case "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake":
		return []string{"extensionsfake"}
	case "github.com/openshift/installer/pkg/asset/installconfig/azure":
		return []string{"icazure"}
	case "github.com/openshift/installer/pkg/types/azure":
		return []string{"azuretypes"}
	case "github.com/coreos/stream-metadata-go/arch":
		return []string{"coreosarch"}
	case "github.com/openshift/installer/pkg/rhcos":
		return []string{"rhcospkg"}
	case "golang.org/x/crypto/ssh":
		return []string{"", "cryptossh"}
	case "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1":
		return []string{"extensionsv1beta1"}
	case "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1":
		return []string{"extensionsv1"}
	case "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset":
		return []string{"extensionsclient"}
	case "k8s.io/apimachinery/pkg/api/errors":
		return []string{"kerrors"}
	case "k8s.io/apimachinery/pkg/runtime":
		return []string{"kruntime"}
	case "k8s.io/apimachinery/pkg/apis/meta/v1":
		return []string{"metav1"}
	case "k8s.io/apimachinery/pkg/runtime/serializer/json":
		return []string{"kjson"}
	case "k8s.io/apimachinery/pkg/util/runtime":
		return []string{"utilruntime"}
	case "k8s.io/apimachinery/pkg/version":
		return []string{"kversion"}
	case "k8s.io/client-go/testing":
		return []string{"ktesting"}
	case "k8s.io/client-go/tools/clientcmd/api/v1":
		return []string{"clientcmdv1"}
	case "k8s.io/client-go/tools/metrics":
		return []string{"kmetrics"}
	case "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1":
		return []string{"azureproviderv1beta1"}
	case "sigs.k8s.io/controller-runtime":
		return []string{"ctrl"}
	}

	return []string{""}
}

func validateImports(path string, fset *token.FileSet, f *ast.File) (errs []error) {
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

nextImport:
	for _, imp := range f.Imports {
		value := strings.Trim(imp.Path.Value, `"`)

		if imp.Name != nil && imp.Name.Name == "." {
			err := validateDotImport(value)
			if err != nil {
				errs = append(errs, err)
			}
			continue
		}

		if imp.Name != nil && imp.Name.Name == "_" {
			err := validateUnderscoreImport(value)
			if err != nil {
				errs = append(errs, err)
			}
			continue
		}

		switch value {
		case "sigs.k8s.io/yaml", "gopkg.in/yaml.v2":
			errs = append(errs, fmt.Errorf("%s is imported; use github.com/ghodss/yaml", value))
			continue nextImport
		case "github.com/google/uuid", "github.com/satori/go.uuid":
			errs = append(errs, fmt.Errorf("%s is imported; use github.com/gofrs/uuid", value))
			continue nextImport
		}

		if strings.HasPrefix(value, "github.com/Azure/azure-sdk-for-go/profiles") {
			errs = append(errs, fmt.Errorf("%s is imported; use github.com/Azure/azure-sdk-for-go/services/*", value))
			continue
		}

		if strings.HasSuffix(value, "/scheme") &&
			value != "k8s.io/client-go/kubernetes/scheme" {
			errs = append(errs, fmt.Errorf("%s is imported; should probably use k8s.io/client-go/kubernetes/scheme", value))
			continue
		}

		if isStandardLibrary(value) {
			if imp.Name != nil {
				errs = append(errs, fmt.Errorf("overridden import %s", value))
			}
			continue
		}

		names := acceptableNames(value)
		if names == nil {
			continue
		}
		for _, name := range names {
			if name == "" && imp.Name == nil ||
				name != "" && imp.Name != nil && imp.Name.Name == name {
				continue nextImport
			}
		}

		errs = append(errs, fmt.Errorf("%s is imported as %q, should be %q", value, imp.Name, names))
	}

	return
}
