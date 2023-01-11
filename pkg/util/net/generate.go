package net

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../util/mocks/$GOPACKAGE
//go:generate go run ../../../vendor/github.com/golang/mock/mockgen -destination=../../../pkg/util/mocks/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/$GOPACKAGE DNSIClient
//go:generate go run ../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../../pkg/util/mocks/$GOPACKAGE/$GOPACKAGE.go
