package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../mocks/ocm/$GOPACKAGE
//go:generate go run ../../../../vendor/github.com/golang/mock/mockgen -destination=../../mocks/ocm/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/ocm/$GOPACKAGE API
//go:generate go run ../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../mocks/ocm/$GOPACKAGE/$GOPACKAGE.go
