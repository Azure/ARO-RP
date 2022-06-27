//go:build tools
// +build tools

package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	_ "github.com/AlekSi/gocov-xml"
	_ "github.com/alvaroloes/enumer"
	_ "github.com/axw/gocov/gocov"
	_ "github.com/go-bindata/go-bindata/go-bindata"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/jewzaam/go-cosmosdb/cmd/gencosmosdb"
	_ "github.com/jstemmer/go-junit-report"
	_ "github.com/openshift/hive/apis"
	_ "golang.org/x/tools/cmd/goimports"
	_ "gotest.tools/gotestsum"
	_ "k8s.io/code-generator/cmd/client-gen"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
