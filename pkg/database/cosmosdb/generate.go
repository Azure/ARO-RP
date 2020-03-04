//go:generate go run ../../../vendor/github.com/jim-minter/go-cosmosdb/cmd/gencosmosdb github.com/Azure/ARO-RP/pkg/api,AsyncOperationDocument github.com/Azure/ARO-RP/pkg/api,BillingDocument github.com/Azure/ARO-RP/pkg/api,MonitorDocument github.com/Azure/ARO-RP/pkg/api,OpenShiftClusterDocument github.com/Azure/ARO-RP/pkg/api,SubscriptionDocument
//go:generate go run ../../../vendor/github.com/golang/mock/mockgen -destination=../../util/mocks/database/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/database/$GOPACKAGE OpenShiftClusterDocumentIterator
//go:generate go run ../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../util/mocks/database/$GOPACKAGE/$GOPACKAGE.go

package cosmosdb

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
