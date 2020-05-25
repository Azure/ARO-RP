package v1alpha1

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../vendor/k8s.io/code-generator/cmd/client-gen --clientset-name versioned --input-base github.com/Azure/ARO-RP/operator --input apis/aro.openshift.io/v1alpha1 --output-package github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset --go-header-file ../hack/licenses/boilerplate.go.txt
//go:generate go run ../vendor/k8s.io/code-generator/cmd/client-gen --clientset-name versioned --input-base github.com/Azure/ARO-RP/operator --input apis/aro.openshift.io/v1alpha1 --output-package github.com/Azure/ARO-RP/pkg/util/aro-operator-client/clientset --go-header-file ../hack/licenses/boilerplate.go.txt
//go:generate gofmt -s -w ../pkg/util/aro-operator-client
//go:generate go run ../vendor/golang.org/x/tools/cmd/goimports -w -local=github.com/Azure/ARO-RP ../pkg/util/aro-operator-client
