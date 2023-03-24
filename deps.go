//go:build tools
// +build tools

package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	_ "github.com/alvaroloes/enumer"
	_ "github.com/go-bindata/go-bindata/go-bindata"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/jewzaam/go-cosmosdb/cmd/gencosmosdb"
	_ "github.com/onsi/ginkgo/v2/ginkgo"
	_ "golang.org/x/tools/cmd/goimports"
	_ "k8s.io/code-generator/cmd/client-gen"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
