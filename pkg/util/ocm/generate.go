package ocm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../../vendor/github.com/golang/mock/mockgen -destination=./api/mock/api.go github.com/Azure/ARO-RP/pkg/util/$GOPACKAGE/api API
//go:generate go run ../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ./api/mock/api.go
