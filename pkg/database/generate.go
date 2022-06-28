package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../vendor/github.com/golang/mock/mockgen -destination=../util/mocks/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/$GOPACKAGE Gateway
//go:generate go run ../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../util/mocks/$GOPACKAGE/$GOPACKAGE.go
