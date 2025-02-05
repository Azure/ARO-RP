package azsecrets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Use source mode to prevent some issues related to generics being present in the interface.
//go:generate rm -rf ../../../../../pkg/util/mocks/azureclient/azuresdk/$GOPACKAGE
//go:generate mockgen -source ./client.go -destination=../../../mocks/azureclient/azuresdk/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/$GOPACKAGE Client
