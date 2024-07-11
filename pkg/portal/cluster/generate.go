package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate ../../../hack/goruntool.sh go-bindata -nometadata -pkg $GOPACKAGE -prefix testdocs testdocs/...
//go:generate gofmt -s -l -w bindata.go
