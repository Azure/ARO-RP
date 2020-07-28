package discovery

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../../vendor/github.com/go-bindata/go-bindata/go-bindata -nometadata -pkg $GOPACKAGE -ignore=generate.go -ignore=bindata.go  -prefix ./cache ./cache/...
//go:generate gofmt -s -l -w ../../../pkg/dynamichelper/discovery/bindata.go
