package armmsi

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Use source mode to prevent some issues related to generics being present in the interface.
//go:generate rm -rf ../../../../../pkg/util/mocks/azureclient/azuresdk/$GOPACKAGE
//go:generate mockgen -source ./federated_identity_credentials.go -destination=../../../mocks/azureclient/azuresdk/$GOPACKAGE/federated_identity_credentials.go github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/$GOPACKAGE FederatedIdentityCredentialsClient
//go:generate mockgen -source ./user_assigned_identities.go -destination=../../../mocks/azureclient/azuresdk/$GOPACKAGE/user_assigned_identities.go github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/$GOPACKAGE UserAssignedIdentitiesClient
