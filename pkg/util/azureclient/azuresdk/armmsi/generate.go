package armmsi

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate rm -rf ../../../../../pkg/util/mocks/azureclient/azuresdk/$GOPACKAGE
//go:generate mockgen -source ./federated_identity_credentials.go -destination=../../../mocks/azureclient/azuresdk/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/$GOPACKAGE FederatedIdentityCredentialsClient
//go:generate goimports -local=github.com/Azure/ARO-RP -e -w ../../../mocks/azureclient/azuresdk/$GOPACKAGE/$GOPACKAGE.go
