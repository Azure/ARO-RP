package machinehealthcheck

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../../../vendor/github.com/go-bindata/go-bindata/go-bindata -nometadata -pkg $GOPACKAGE -prefix staticresources staticresources/...
//go:generate gofmt -s -l -w bindata.go
