package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../../util/mocks/azureclient/$GOPACKAGE
//go:generate ../../../../hack/goruntool.sh mockgen -destination=../../../util/mocks/azureclient/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/$GOPACKAGE BaseClient
//go:generate ../../../../hack/goruntool.sh goimports -local=github.com/Azure/ARO-RP -e -w ../../../util/mocks/azureclient/$GOPACKAGE/$GOPACKAGE.go
