package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../util/mocks/operator/$GOPACKAGE
//go:generate go run ../../../vendor/github.com/golang/mock/mockgen -destination=../../util/mocks/operator/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/operator/$GOPACKAGE Operator
//go:generate go run ../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../util/mocks/operator/$GOPACKAGE/$GOPACKAGE.go
