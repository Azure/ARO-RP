package muo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// bindata for the above yaml files
//go:generate go run ../../../../vendor/github.com/go-bindata/go-bindata/go-bindata -nometadata -pkg muo -prefix staticresources/ -o bindata.go staticresources/...
//go:generate gofmt -s -l -w bindata.go

//go:generate rm -rf ../../mocks/$GOPACKAGE
//go:generate go run ../../../../vendor/github.com/golang/mock/mockgen -destination=../../mocks/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/operator/controllers/$GOPACKAGE Deployer
//go:generate go run ../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../mocks/$GOPACKAGE/$GOPACKAGE.go
