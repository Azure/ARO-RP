package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../vendor/k8s.io/code-generator/cmd/client-gen --clientset-name versioned --input-base ../../vendor/github.com/openshift/hive/apis --input "hive/v1" -o ./ --trim-path-prefix github.com/Azure/ARO-RP/pkg/hive/ --output-package github.com/Azure/ARO-RP/pkg/hive/clientset --go-header-file ../../hack/licenses/boilerplate.go.txt -v 1
//go:generate gofmt -s -w ./clientset
//go:generate go run ../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ./clientset
