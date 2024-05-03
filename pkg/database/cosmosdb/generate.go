package cosmosdb

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// removed generation from github.com/jewzaam/go-cosmosdb/ for time being as I have manually added the code. Waiting for https://github.com/jewzaam/go-cosmosdb/pull/22 to merge.

//go:generate go run ../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ./
//go:generate go run ../../../vendor/github.com/golang/mock/mockgen -destination=../../util/mocks/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/database/$GOPACKAGE PermissionClient
//go:generate go run ../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../util/mocks/$GOPACKAGE/$GOPACKAGE.go
