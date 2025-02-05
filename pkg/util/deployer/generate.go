package deployer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate mockgen -destination=../mocks/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/$GOPACKAGE Deployer
