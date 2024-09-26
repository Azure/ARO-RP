package armcosmos

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../../../../pkg/util/mocks/azureclient/azuresdk/$GOPACKAGE
//go:generate mockgen -destination=../../../mocks/azureclient/azuresdk/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/$GOPACKAGE DatabaseAccountsClient
//go:generate goimports -local=github.com/Azure/ARO-RP -e -w ../../../mocks/azureclient/azuresdk/$GOPACKAGE/$GOPACKAGE.go
