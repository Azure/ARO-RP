package armstorage

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../../../util/mocks/$GOPACKAGE
//go:generate mockgen -destination=../../../../util/mocks/azureclient/azuresdk/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/$GOPACKAGE AccountsClient
//go:generate goimports -local=github.com/Azure/ARO-RP -e -w ../../../../util/mocks/azureclient/azuresdk/$GOPACKAGE/$GOPACKAGE.go
